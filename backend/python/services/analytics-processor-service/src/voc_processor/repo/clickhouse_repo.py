import asyncio
import json
from typing import List, Dict, Any
from tenacity import retry, wait_exponential, stop_after_attempt
from datetime import datetime

from clickhouse_connect import get_client


class ClickHouseRepo:
    def __init__(self, settings):
        self.settings = settings
        self.client = get_client(
            host=settings.clickhouse_host,
            port=settings.clickhouse_port,
            username=settings.clickhouse_user,
            password=settings.clickhouse_password,
            database=settings.clickhouse_db,
        )

    async def ensure_table(self):
        ddl = """
        CREATE TABLE IF NOT EXISTS voc_events (
            event_id UUID,
            tenant_id String,
            event_type String,
            ts DateTime64(3),
            payload String,
            sentiment Nullable(Float32),
            topic Nullable(String),
            processed_at DateTime64(3)
        ) ENGINE = MergeTree()
        PARTITION BY (tenant_id, toYYYYMM(ts))
        ORDER BY (tenant_id, ts, event_id)
        """
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self.client.command, ddl)

    @retry(wait=wait_exponential(min=1, max=10), stop=stop_after_attempt(5))
    async def insert_events(self, events: List[Dict[str, Any]]):
        if not events:
            return
        rows = []
        for e in events:
            rows.append((
                str(e.get("event_id")),
                e.get("tenant_id"),
                e.get("event_type"),
                e.get("ts"),
                json.dumps(e.get("payload", {})),
                e.get("sentiment"),
                e.get("topic"),
                e.get("processed_at") or datetime.utcnow(),
            ))
        columns = ["event_id", "tenant_id", "event_type", "ts", "payload", "sentiment", "topic", "processed_at"]
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self.client.insert, "voc_events", rows, columns)
