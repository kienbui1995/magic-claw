"""Embedded MagiC server — downloads and runs the Go binary in-process."""

import os
import platform
import stat
import subprocess
import time
from pathlib import Path

import httpx

from magic_ai_sdk.client import MagiCClient

REPO = "kienbui1995/magic"
CACHE_DIR = Path.home() / ".magic" / "bin"


def _detect_platform() -> str:
    system = platform.system().lower()
    machine = platform.machine().lower()

    if system not in ("linux", "darwin"):
        raise RuntimeError(f"Embedded mode not supported on {system}. Run the server manually: https://github.com/{REPO}")

    if machine in ("x86_64", "amd64"):
        arch = "amd64"
    elif machine in ("arm64", "aarch64"):
        arch = "arm64"
    else:
        raise RuntimeError(f"Unsupported architecture: {machine}")

    return f"magic-{system}-{arch}"


def _binary_path(version: str = "latest") -> Path:
    name = _detect_platform()
    return CACHE_DIR / version / name


def _download_binary(version: str = "latest") -> Path:
    name = _detect_platform()
    dest = _binary_path(version)
    dest.parent.mkdir(parents=True, exist_ok=True)

    if version == "latest":
        url = f"https://github.com/{REPO}/releases/latest/download/{name}"
    else:
        url = f"https://github.com/{REPO}/releases/download/{version}/{name}"

    print(f"Downloading MagiC binary from {url} ...")
    with httpx.Client(follow_redirects=True, timeout=60) as client:
        response = client.get(url)
        response.raise_for_status()
        dest.write_bytes(response.content)

    dest.chmod(dest.stat().st_mode | stat.S_IEXEC | stat.S_IXGRP | stat.S_IXOTH)
    return dest


def _get_binary(version: str = "latest") -> Path:
    path = _binary_path(version)
    if path.exists():
        return path
    return _download_binary(version)


class MagiC:
    """Embedded MagiC server. Use as a context manager.

    Example::

        with MagiC() as client:
            client.submit_task({"type": "greet", "input": {}})
    """

    def __init__(
        self,
        port: int = 18080,
        api_key: str = "",
        version: str = "latest",
        timeout: float = 10.0,
    ):
        self._port = port
        self._api_key = api_key
        self._version = version
        self._timeout = timeout
        self._proc: subprocess.Popen | None = None

    def start(self) -> "MagiCClient":
        binary = _get_binary(self._version)
        env = os.environ.copy()
        env["MAGIC_PORT"] = str(self._port)
        if self._api_key:
            env["MAGIC_API_KEY"] = self._api_key

        self._proc = subprocess.Popen(
            [str(binary), "serve"],
            env=env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        self._wait_ready()
        return MagiCClient(
            base_url=f"http://localhost:{self._port}",
            api_key=self._api_key,
        )

    def stop(self) -> None:
        if self._proc and self._proc.poll() is None:
            self._proc.terminate()
            try:
                self._proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self._proc.kill()
        self._proc = None

    def _wait_ready(self) -> None:
        deadline = time.monotonic() + self._timeout
        url = f"http://localhost:{self._port}/health"
        while time.monotonic() < deadline:
            if self._proc.poll() is not None:
                raise RuntimeError("MagiC server exited unexpectedly")
            try:
                with httpx.Client(timeout=1) as c:
                    if c.get(url).is_success:
                        return
            except httpx.TransportError:
                pass
            time.sleep(0.2)
        raise TimeoutError(f"MagiC server did not start within {self._timeout}s")

    def __enter__(self) -> "MagiCClient":
        return self.start()

    def __exit__(self, *_) -> None:
        self.stop()
