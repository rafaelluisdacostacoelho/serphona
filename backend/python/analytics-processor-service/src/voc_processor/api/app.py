import asyncio
from fastapi import FastAPI, Query
from typing import Optional
from ..repo.clickhouse_repo import ClickHouseRepo
from ..config import settings

def create_app():
    app = FastAPI(title="voc-processor")
    repo = ClickHouseRepo(settings)

    @app.on_event("startup")
    async def startup():
        # ensure table exists
        await repo.ensure_table()

    @app.get("/healthz")
    async def healthz():
        return {"status": "ok"}

    @app.get("/metrics/summary")
    async def summary(tenant_id: Optional[str] = Query(None), days: int = 7):
        tenant_filter = ""
        if tenant_id:
            tenant_filter = f"AND tenant_id = '{tenant_id}'"
        sql = f"""
        SELECT event_type, count() AS cnt, avg(sentiment) as avg_sent
        FROM voc_events
        WHERE ts >= now() - INTERVAL {days} DAY
        {tenant_filter}
        GROUP BY event_type
        """
        loop = asyncio.get_event_loop()
        rows = await loop.run_in_executor(None, repo.client.query, sql)
        return {"rows": rows.result_rows}

    return app
