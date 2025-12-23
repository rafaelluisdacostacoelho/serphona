import asyncio
import signal
import sys
from uvicorn import Config, Server

from .config import settings
from .repo.clickhouse_repo import ClickHouseRepo
from .worker import ConsumerWorker
from .api.app import create_app


async def run():
    repo = ClickHouseRepo(settings)
    await repo.ensure_table()

    worker = ConsumerWorker(repo)
    await worker.start()

    app = create_app()
    config = Config(app, host="0.0.0.0", port=settings.http_port, loop="asyncio")
    server = Server(config)
    server_task = asyncio.create_task(server.serve())

    # best-effort signal handling; on Windows loop.add_signal_handler may not be available
    stop_event = asyncio.Event()

    try:
        loop = asyncio.get_event_loop()
        try:
            loop.add_signal_handler(signal.SIGTERM, stop_event.set)
            loop.add_signal_handler(signal.SIGINT, stop_event.set)
        except NotImplementedError:
            # Windows fallback: rely on KeyboardInterrupt
            pass

        await stop_event.wait()
    except KeyboardInterrupt:
        pass

    await worker.stop()
    await repo.client.close()
    # shutdown uvicorn server
    server.should_exit = True
    await server_task


if __name__ == "__main__":
    asyncio.run(run())
