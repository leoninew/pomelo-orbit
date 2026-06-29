#!/usr/bin/env python3
"""Start frontend and backend development servers together."""

from __future__ import annotations

import logging
import os
import signal
import shutil
import subprocess
import sys
import time
from pathlib import Path
from types import FrameType
from typing import Any, cast

ROOT_DIR = Path(__file__).resolve().parent.parent
logger = logging.getLogger(__name__)
shutdown_signal: int | None = None


def command_path(command: str) -> str:
    """Return executable path for command on the current platform."""
    path = shutil.which(command)
    assert path, f"未找到命令: {command}"
    return path


class DevProcess:
    """Development server process."""

    def __init__(
        self,
        name: str,
        cwd: Path,
        command: list[str],
        env: dict[str, str] | None = None,
    ):
        self.name = name
        self.cwd = cwd
        self.command = command
        self.env = env
        self.process: subprocess.Popen[bytes] | None = None

    def start(self) -> None:
        """Start process in its own process group."""
        logger.info("启动%s...", self.name)
        kwargs: dict[str, Any] = {
            "cwd": self.cwd,
            "env": self.env,
        }

        if os.name == "nt":
            kwargs["creationflags"] = subprocess.CREATE_NEW_PROCESS_GROUP
        else:
            kwargs["start_new_session"] = True

        self.process = subprocess.Popen(self.command, **kwargs)

    def stop(self) -> None:
        """Stop process and its children."""
        if self.process is None or self.process.poll() is not None:
            return

        logger.info("停止%s...", self.name)
        if os.name == "nt":
            self.process.send_signal(signal.CTRL_BREAK_EVENT)
        else:
            killpg = cast(Any, os).killpg
            killpg(self.process.pid, signal.SIGTERM)

    def kill(self) -> None:
        """Force kill process when graceful stop times out."""
        if self.process is None or self.process.poll() is not None:
            return

        if os.name == "nt":
            self.kill_process_tree(force=True)
        else:
            killpg = cast(Any, os).killpg
            sigkill = cast(Any, signal).SIGKILL
            killpg(self.process.pid, sigkill)

    def kill_process_tree(self, force: bool) -> None:
        """Kill the Windows process tree started by this process."""
        assert self.process is not None
        command = ["taskkill", "/PID", str(self.process.pid), "/T"]
        if force:
            command.append("/F")
        subprocess.run(
            command, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False
        )


def stop_all(processes: list[DevProcess]) -> None:
    """Stop all dev processes."""
    for process in processes:
        process.stop()

    deadline = time.monotonic() + 10.0
    for process in processes:
        if process.process is None:
            continue
        remaining_seconds = deadline - time.monotonic()
        if remaining_seconds <= 0:
            process.kill()
            continue
        try:
            process.process.wait(timeout=remaining_seconds)
        except subprocess.TimeoutExpired:
            process.kill()

    force_kill_deadline = time.monotonic() + 5.0
    for process in processes:
        if process.process is None:
            continue
        remaining_seconds = force_kill_deadline - time.monotonic()
        if remaining_seconds <= 0:
            continue
        try:
            process.process.wait(timeout=remaining_seconds)
        except subprocess.TimeoutExpired:
            pass


def handle_shutdown_signal(signum: int, _frame: FrameType | None) -> None:
    """Record shutdown signal for the main loop."""
    global shutdown_signal
    shutdown_signal = signum


def wait_any(processes: list[DevProcess]) -> int:
    """Wait until any process exits, then return its exit code."""
    while True:
        if shutdown_signal is not None:
            return 130 if shutdown_signal == signal.SIGINT else 143

        for process in processes:
            assert process.process is not None
            exit_code = process.process.poll()
            if exit_code is not None:
                return exit_code

        time.sleep(0.2)


def main() -> int:
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    signal.signal(signal.SIGINT, handle_shutdown_signal)
    signal.signal(signal.SIGTERM, handle_shutdown_signal)

    processes = [
        DevProcess(
            name="后端服务",
            cwd=ROOT_DIR / "backend-go",
            command=[command_path("air"), "-c", ".air.api.toml"],
        ),
        DevProcess(
            name="前端服务",
            cwd=ROOT_DIR / "frontend",
            command=[command_path("yarn"), "dev"],
        ),
    ]

    try:
        for process in processes:
            process.start()
        logger.info("前端、后端 API 与后台任务热重载已启动，按 Ctrl+C 退出。")
        exit_code = wait_any(processes)
        if shutdown_signal is not None:
            logger.info("收到退出信号，正在停止开发服务器...")
        else:
            logger.info("开发服务器已退出，正在停止剩余进程...")
        return exit_code
    except KeyboardInterrupt:
        logger.info("收到退出信号，正在停止开发服务器...")
        return 130
    finally:
        stop_all(processes)


if __name__ == "__main__":
    sys.exit(main())
