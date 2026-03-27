"""Tests for embedded MagiC — unit tests only, no real download/subprocess."""

from unittest.mock import patch

import pytest

from magic_ai_sdk.embedded import _detect_platform


def test_detect_platform_linux_amd64():
    with patch("platform.system", return_value="Linux"), patch("platform.machine", return_value="x86_64"):
        assert _detect_platform() == "magic-linux-amd64"


def test_detect_platform_darwin_arm64():
    with patch("platform.system", return_value="Darwin"), patch("platform.machine", return_value="arm64"):
        assert _detect_platform() == "magic-darwin-arm64"


def test_detect_platform_windows_raises():
    with patch("platform.system", return_value="Windows"):
        with pytest.raises(RuntimeError, match="not supported"):
            _detect_platform()


def test_magic_exports():
    from magic_ai_sdk import MagiC

    assert MagiC is not None
