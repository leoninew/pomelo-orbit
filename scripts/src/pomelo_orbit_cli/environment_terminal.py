"""Cross-platform raw terminal adapter for the shared orbit-terminal-v1 protocol."""

from __future__ import annotations

import asyncio
import codecs
import json
import logging
import sys
from contextlib import ExitStack
from typing import Any

from prompt_toolkit.input import create_input
from prompt_toolkit.input.ansi_escape_sequences import REVERSE_ANSI_SEQUENCES
from prompt_toolkit.key_binding import KeyPress
from prompt_toolkit.keys import Keys
from prompt_toolkit.output import create_output
from websockets.exceptions import ConnectionClosed, InvalidStatus, WebSocketException
from websockets.sync.client import connect as websocket_connect
from websockets.sync.connection import Connection
from websockets.typing import Subprotocol

from pomelo_orbit_cli.environment_client import (
    EnvironmentClient,
    EnvironmentError,
    api_error,
)


def key_data(key: KeyPress) -> str:
    if key.key == Keys.BracketedPaste:
        return "\x1b[200~" + key.data + "\x1b[201~"
    return key.data or (
        REVERSE_ANSI_SEQUENCES.get(key.key, "") if isinstance(key.key, Keys) else ""
    )


async def run_terminal(ws: Connection, terminal_input: Any, output: Any) -> int:
    loop = asyncio.get_running_loop()
    queue: asyncio.Queue[bytes | str] = asyncio.Queue(maxsize=64)
    stop: asyncio.Future[int] = loop.create_future()
    ready = asyncio.Event()
    decoder = codecs.getincrementaldecoder("utf-8")(errors="replace")
    flush_handle: asyncio.TimerHandle | None = None

    def enqueue(data: bytes | str) -> None:
        try:
            queue.put_nowait(data)
        except asyncio.QueueFull:
            if not stop.done():
                stop.set_exception(
                    EnvironmentError(
                        "input_overflow", "Terminal input queue exceeded its limit"
                    )
                )

    def accept_keys(keys: list[KeyPress]) -> None:
        for key in keys:
            data = key_data(key)
            if data == "\x1d":
                if not stop.done():
                    stop.set_result(0)
                return
            if data:
                encoded = data.encode("utf-8")
                for offset in range(0, len(encoded), 16 * 1024):
                    enqueue(encoded[offset : offset + 16 * 1024])

    def flush_keys() -> None:
        accept_keys(terminal_input.flush_keys())

    def input_ready() -> None:
        nonlocal flush_handle
        if flush_handle is not None:
            flush_handle.cancel()
        accept_keys(terminal_input.read_keys())
        if terminal_input.closed and not stop.done():
            stop.set_exception(
                EnvironmentError("input_closed", "Local terminal input closed")
            )
        flush_handle = loop.call_later(0.1, flush_keys)

    async def receive() -> int:
        while True:
            message = await asyncio.to_thread(ws.recv)
            if isinstance(message, bytes):
                output.write_raw(decoder.decode(message))
                output.flush()
                continue
            try:
                control = json.loads(message)
                kind = control["type"]
                if kind == "ready":
                    ready.set()
                elif kind == "exit":
                    code = control.get("code", 0)
                    if type(code) is not int:
                        raise ValueError
                    return code if 0 <= code <= 255 else 255
                else:
                    raise EnvironmentError(
                        "terminal_closed",
                        f"Terminal ended: {kind}"
                        if kind
                        in {
                            "connection_failed",
                            "idle_timeout",
                            "access_or_target_changed",
                            "session_limit",
                        }
                        else "Terminal protocol failed",
                    )
            except (ValueError, KeyError, TypeError):
                raise EnvironmentError(
                    "contract_mismatch", "Invalid terminal control frame"
                ) from None

    async def send() -> None:
        await ready.wait()
        while True:
            await asyncio.to_thread(ws.send, await queue.get())

    async def resize() -> None:
        await ready.wait()
        previous = None
        while True:
            size = output.get_size()
            current = (min(500, max(1, size.columns)), min(200, max(1, size.rows)))
            if current != previous:
                enqueue(
                    json.dumps(
                        {"type": "resize", "cols": current[0], "rows": current[1]}
                    )
                )
                previous = current
            await asyncio.sleep(0.25)

    tasks: list[asyncio.Task[Any]] = []
    with ExitStack() as stack:
        stack.enter_context(terminal_input.raw_mode())
        stack.enter_context(terminal_input.attach(input_ready))
        try:
            tasks = [
                asyncio.create_task(receive()),
                asyncio.create_task(send()),
                asyncio.create_task(resize()),
            ]
            done, _ = await asyncio.wait(
                [*tasks, stop], return_when=asyncio.FIRST_COMPLETED
            )
            if stop in done:
                return stop.result()
            for task in tasks:
                if task in done:
                    return task.result()
            raise EnvironmentError("terminal_closed", "Terminal stopped unexpectedly")
        finally:
            if flush_handle is not None:
                flush_handle.cancel()
            # Close before cancelling to_thread workers: cancellation alone cannot
            # interrupt a blocking recv/send or release its executor thread.
            await asyncio.to_thread(ws.close)
            for task in tasks:
                task.cancel()
            await asyncio.gather(*tasks, return_exceptions=True)
            if not stop.done():
                stop.cancel()
            elif not stop.cancelled():
                stop.exception()
            output.write_raw(decoder.decode(b"", final=True))
            output.flush()


def connect(client: EnvironmentClient, project: str) -> int:
    if sys.platform == "win32":
        from prompt_toolkit.output.windows10 import is_win_vt100_enabled

        if not is_win_vt100_enabled():
            raise EnvironmentError(
                "terminal_unavailable", "Use a Windows console with VT support"
            )
    # Isolated logger prevents --verbose from logging PAT headers or frames.
    quiet_logger = logging.Logger(
        "orbit.environment.websocket", level=logging.CRITICAL + 1
    )
    quiet_logger.propagate = False
    try:
        with websocket_connect(
            client.terminal_url(project),
            additional_headers={
                "Authorization": f"Bearer {client.settings.access_token}"
            },
            subprotocols=[Subprotocol("orbit-terminal-v1")],
            proxy=None,
            compression=None,
            open_timeout=15,
            close_timeout=5,
            max_size=64 * 1024,
            ping_interval=20,
            ping_timeout=20,
            logger=quiet_logger,
        ) as ws:
            if ws.subprotocol != "orbit-terminal-v1":
                raise EnvironmentError(
                    "contract_mismatch", "Terminal protocol was not negotiated"
                )
            terminal_input = create_input(stdin=sys.stdin)
            try:
                return asyncio.run(
                    run_terminal(ws, terminal_input, create_output(stdout=sys.stdout))
                )
            finally:
                terminal_input.close()
    except InvalidStatus as error:
        raise api_error(
            error.response.status_code, bytes(error.response.body)
        ) from None
    except (ConnectionClosed, WebSocketException, OSError, TimeoutError):
        raise EnvironmentError(
            "transport_error", "Terminal connection lost or unavailable"
        ) from None
