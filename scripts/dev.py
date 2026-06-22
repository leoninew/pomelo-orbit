#!/usr/bin/env python3
"""Start frontend and backend development servers together."""

from __future__ import annotations

import os
import signal
import shutil
import subprocess
import sys
import time
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent


def command_path(command: str) -> str:
    """Return executable path for command on the current platform."""
    path = shutil.which(command)
    assert path, f"未找到命令: {command}"
    return path


class DevProcess:
    """Development server process."""

    def __init__(self, name: str, cwd: Path, command: list[str], env: dict[str, str] | None = None):
        self.name = name
        self.cwd = cwd
        self.command = command
        self.env = env
        self.process: subprocess.Popen[bytes] | None = None

    def start(self) -> None:
        """Start process in its own process group."""
        print(f"启动{self.name}...", flush=True)
        kwargs: dict[str, object] = {
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

        print(f"停止{self.name}...", flush=True)
        if os.name == "nt":
            self.kill_process_tree(force=False)
        else:
            os.killpg(self.process.pid, signal.SIGTERM)

    def kill(self) -> None:
        """Force kill process when graceful stop times out."""
        if self.process is None or self.process.poll() is not None:
            return

        if os.name == "nt":
            self.kill_process_tree(force=True)
        else:
            os.killpg(self.process.pid, signal.SIGKILL)

    def kill_process_tree(self, force: bool) -> None:
        """Kill the Windows process tree started by this process."""
        assert self.process is not None
        command = ["taskkill", "/PID", str(self.process.pid), "/T"]
        if force:
            command.append("/F")
        subprocess.run(command, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)


def stop_all(processes: list[DevProcess]) -> None:
    """Stop all dev processes."""
    for process in reversed(processes):
        process.stop()

    for process in processes:
        if process.process is None:
            continue
        try:
            process.process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()


def wait_any(processes: list[DevProcess]) -> int:
    """Wait until any process exits, then return its exit code."""
    while True:
        for process in processes:
            assert process.process is not None
            exit_code = process.process.poll()
            if exit_code is not None:
                return exit_code

        time.sleep(1)


def main() -> int:
    processes = [
        DevProcess(
            name="后端服务",
            cwd=ROOT_DIR / "backend-go",
            command=[command_path("air"), "-c", ".air.api.toml"],
        ),
        DevProcess(
            name="后端 Worker",
            cwd=ROOT_DIR / "backend-go",
            command=[command_path("air"), "-c", ".air.worker.toml"],
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
        print("前端、后端 API hot reload 和 Worker hot reload 已启动，按 Ctrl+C 退出。", flush=True)
        exit_code = wait_any(processes)
        print("开发服务器已退出，正在停止剩余进程...", flush=True)
        return exit_code
    except KeyboardInterrupt:
        print("收到退出信号，正在停止开发服务器...", flush=True)
        return 130
    finally:
        stop_all(processes)


if __name__ == "__main__":
    sys.exit(main())
