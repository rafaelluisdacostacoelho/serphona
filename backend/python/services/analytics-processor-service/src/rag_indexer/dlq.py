import json
import os
from typing import Any, Dict


class DLQWriter:
    """Minimal DLQ writer that appends JSON lines to a file path."""

    def __init__(self, path: str):
        self.path = path

    async def publish(self, payload: Dict[str, Any]):
        if not self.path:
            return
        os.makedirs(os.path.dirname(self.path), exist_ok=True)
        with open(self.path, "a", encoding="utf-8") as fh:
            fh.write(json.dumps(payload, default=str))
            fh.write("\n")
