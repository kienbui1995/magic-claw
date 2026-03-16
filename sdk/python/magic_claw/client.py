import httpx

class MagiCClient:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip("/")
        self._client = httpx.Client(base_url=self.base_url, timeout=30)

    def register_worker(self, payload: dict) -> dict:
        resp = self._client.post("/api/v1/workers/register", json=payload)
        resp.raise_for_status()
        return resp.json()

    def heartbeat(self, worker_id: str, current_load: int = 0) -> dict:
        resp = self._client.post("/api/v1/workers/heartbeat", json={
            "worker_id": worker_id,
            "current_load": current_load,
            "status": "active",
        })
        resp.raise_for_status()
        return resp.json()

    def health(self) -> dict:
        resp = self._client.get("/health")
        resp.raise_for_status()
        return resp.json()
