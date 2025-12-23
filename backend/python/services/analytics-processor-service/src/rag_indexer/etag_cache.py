import json
import os
from typing import Dict, Optional


class ETagCache:
    """Simple JSON-backed ETag cache keyed by URI."""

    def __init__(self, path: Optional[str]):
        self.path = path
        self._data: Dict[str, str] = {}
        if path:
            self._load()

    def _load(self):
        if not self.path or not os.path.exists(self.path):
            return
        try:
            with open(self.path, "r", encoding="utf-8") as fh:
                self._data = json.load(fh)
        except Exception:
            self._data = {}

    def get(self, uri: str) -> Optional[str]:
        return self._data.get(uri)

    def set(self, uri: str, etag: Optional[str]):
        if not self.path or not etag:
            return
        self._data[uri] = etag
        try:
            os.makedirs(os.path.dirname(self.path), exist_ok=True)
            with open(self.path, "w", encoding="utf-8") as fh:
                json.dump(self._data, fh)
        except Exception:
            pass
