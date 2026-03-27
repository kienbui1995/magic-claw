"""MagiC Python SDK — Build AI workers for the MagiC framework."""

from magic_ai_sdk.client import AsyncMagiCClient, MagiCClient
from magic_ai_sdk.decorators import capability
from magic_ai_sdk.embedded import MagiC
from magic_ai_sdk.worker import Worker

__version__ = "0.2.0"
__all__ = ["Worker", "MagiCClient", "AsyncMagiCClient", "MagiC", "capability", "__version__"]
