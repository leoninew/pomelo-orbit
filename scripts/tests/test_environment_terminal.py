from __future__ import annotations

import asyncio
import json
import queue
from contextlib import contextmanager
from typing import Any, Iterator

import pytest
from prompt_toolkit.data_structures import Size
from prompt_toolkit.key_binding import KeyPress
from prompt_toolkit.keys import Keys

from pomelo_orbit_cli.environment_client import EnvironmentError
from pomelo_orbit_cli.environment_terminal import key_data, run_terminal


def test_terminal_key_translation_preserves_controls_and_paste() -> None:
    assert key_data(KeyPress(Keys.ControlC, "\x03")) == "\x03"
    assert key_data(KeyPress(Keys.ControlD, "\x04")) == "\x04"
    assert key_data(KeyPress(Keys.Up, "")) == "\x1b[A"
    assert key_data(KeyPress(Keys.BracketedPaste, "a\nb")) == "\x1b[200~a\nb\x1b[201~"


class TerminalInput:
    closed = False
    restored = False
    detached = False
    callback: Any

    def __init__(self, keys: list[KeyPress]):
        self.keys = keys

    @contextmanager
    def raw_mode(self) -> Iterator[None]:
        try:
            yield
        finally:
            self.restored = True

    @contextmanager
    def attach(self, callback: Any) -> Iterator[None]:
        self.callback = callback
        try:
            yield
        finally:
            self.detached = True

    def read_keys(self) -> list[KeyPress]:
        keys, self.keys = self.keys, []
        return keys

    def flush_keys(self) -> list[KeyPress]:
        return []


class TerminalOutput:
    def __init__(self) -> None:
        self.text = ""

    def write_raw(self, text: str) -> None:
        self.text += text

    def flush(self) -> None:
        pass

    def get_size(self) -> Size:
        return Size(rows=40, columns=120)


class Socket:
    def __init__(self) -> None:
        self.frames: queue.Queue[bytes | str] = queue.Queue()
        self.sent: list[bytes | str] = []
        self.closed = False

    def recv(self) -> bytes | str:
        return self.frames.get(timeout=5)

    def send(self, data: bytes | str) -> None:
        self.sent.append(data)

    def close(self) -> None:
        self.closed = True
        self.frames.put('{"type":"exit","code":0}')


@pytest.mark.parametrize("detach", [False, True])
def test_terminal_controls_resize_utf8_and_cleanup(detach: bool) -> None:
    async def scenario() -> None:
        ws = Socket()
        inp = TerminalInput(
            [KeyPress(Keys.ControlSquareClose, "\x1d")]
            if detach
            else [KeyPress(Keys.ControlC, "\x03"), KeyPress(Keys.ControlD, "\x04")]
        )
        out = TerminalOutput()
        ws.frames.put('{"type":"ready"}')
        ws.frames.put(b"\xe4\xb8")
        ws.frames.put(b"\xad")
        task = asyncio.create_task(run_terminal(ws, inp, out))  # type: ignore[arg-type]
        for _ in range(200):
            if ws.sent:
                break
            await asyncio.sleep(0.005)
        inp.callback()
        if not detach:
            for _ in range(200):
                if b"\x04" in ws.sent:
                    break
                await asyncio.sleep(0.005)
            ws.frames.put('{"type":"exit","code":7}')
        assert await asyncio.wait_for(task, 3) == (0 if detach else 7)
        assert out.text == "中"
        assert any(
            isinstance(frame, str)
            and json.loads(frame) == {"type": "resize", "cols": 120, "rows": 40}
            for frame in ws.sent
        )
        if not detach:
            assert b"\x03" in ws.sent and b"\x04" in ws.sent
        assert ws.closed and inp.restored and inp.detached

    asyncio.run(scenario())


def test_terminal_protocol_failure_restores_input() -> None:
    async def scenario() -> None:
        ws, inp, out = Socket(), TerminalInput([]), TerminalOutput()
        ws.frames.put('{"type":"connection_failed"}')
        with pytest.raises(EnvironmentError):
            await run_terminal(ws, inp, out)  # type: ignore[arg-type]
        assert ws.closed and inp.restored and inp.detached

    asyncio.run(scenario())


def test_terminal_paste_queue_overflow_closes_visibly() -> None:
    async def scenario() -> None:
        ws, inp, out = (
            Socket(),
            TerminalInput([KeyPress(Keys.BracketedPaste, "x" * (2 * 1024 * 1024))]),
            TerminalOutput(),
        )
        ws.frames.put('{"type":"ready"}')
        task = asyncio.create_task(run_terminal(ws, inp, out))  # type: ignore[arg-type]
        await asyncio.sleep(0.01)
        inp.callback()
        with pytest.raises(EnvironmentError, match="input_overflow"):
            await asyncio.wait_for(task, 3)
        assert ws.closed and inp.restored
        assert all(
            len(frame) <= 16 * 1024 for frame in ws.sent if isinstance(frame, bytes)
        )

    asyncio.run(scenario())
