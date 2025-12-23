import asyncio
import logging
from aiokafka import AIOKafkaConsumer
from prometheus_client import start_http_server

from .worker import RAGWorker
from .repo.pgvector_repo import PGVectorRepo
from .config import settings

logging.basicConfig(level=logging.INFO)


def make_consumer(bootstrap: str, group_id: str, topics):
    return AIOKafkaConsumer(
        *topics,
        bootstrap_servers=bootstrap,
        group_id=group_id,
        enable_auto_commit=False,
        value_deserializer=lambda m: m,
    )


async def main_async():
    start_http_server(settings.prometheus_port)
    repo = await PGVectorRepo.create(
        settings.pg_dsn,
        table=settings.pg_table,
        min_size=settings.pg_pool_min,
        max_size=settings.pg_pool_max,
        expect_dim=settings.embedding_dim,
    )
    worker = RAGWorker(repo, make_consumer)

    await worker.start()
    try:
        while True:
            await asyncio.sleep(1)
    except (KeyboardInterrupt, asyncio.CancelledError):
        pass
    finally:
        await worker.stop()


def main():
    asyncio.run(main_async())


if __name__ == "__main__":
    main()
