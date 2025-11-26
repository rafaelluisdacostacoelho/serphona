import asyncio
import json
import time
from typing import List
from aiokafka.structs import TopicPartition

from .kafka_client import make_consumer
from .models.events import BaseEvent
from .nlp.pipeline import annotate_nlp
from .repo.clickhouse_repo import ClickHouseRepo
from .config import settings
from .utils.metrics import M_CONSUMED, M_PROCESSED, M_PROCESS_LATENCY

BATCH_SIZE = 500
BATCH_FLUSH_SECONDS = 2.0


class ConsumerWorker:
    def __init__(self, repo: ClickHouseRepo):
        self.repo = repo
        self._stopping = asyncio.Event()
        self.consumer = None

    async def start(self):
        self.consumer = make_consumer(settings.kafka_bootstrap, settings.kafka_group_id, settings.kafka_topics)
        await self.consumer.start()
        self._task = asyncio.create_task(self._run())

    async def stop(self):
        self._stopping.set()
        if getattr(self, "_task", None):
            await self._task
        if self.consumer:
            await self.consumer.stop()

    async def _run(self):
        batch = []
        last_flush = time.monotonic()
        try:
            async for msg in self.consumer:
                try:
                    data = json.loads(msg.value)
                    event = BaseEvent.parse_obj(data)
                except Exception:
                    # TODO: push to DLQ
                    continue

                batch.append((msg, event.dict()))
                M_CONSUMED.inc()
                now = time.monotonic()
                if len(batch) >= BATCH_SIZE or (now - last_flush) >= BATCH_FLUSH_SECONDS:
                    await self._process_batch(batch)
                    batch = []
                    last_flush = now

                if self._stopping.is_set():
                    break

            if batch:
                await self._process_batch(batch)
        finally:
            pass

    async def _process_batch(self, batch: List):
        start = time.monotonic()
        msgs, events = zip(*batch)
        # events are dicts
        annotated = await annotate_nlp(list(events))
        await self.repo.insert_events(annotated)

        # commit offsets: highest offset per partition
        offsets = {}
        for msg in msgs:
            tp = (msg.topic, msg.partition)
            offsets[tp] = msg.offset

        commit_map = {}
        for (topic, part), offset in offsets.items():
            commit_map[TopicPartition(topic, part)] = offset + 1
        await self.consumer.commit(commit_map)

        M_PROCESSED.inc(len(annotated))
        M_PROCESS_LATENCY.observe(time.monotonic() - start)
