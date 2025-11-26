from pydantic import BaseSettings, Field
from typing import List

class Settings(BaseSettings):
    # Kafka
    kafka_bootstrap: str = Field("localhost:9092", env="KAFKA_BOOTSTRAP")
    kafka_topics: List[str] = Field(["voc-events"], env="KAFKA_TOPICS")
    kafka_group_id: str = Field("voc-processor-group", env="KAFKA_GROUP_ID")

    # ClickHouse
    clickhouse_host: str = Field("localhost", env="CLICKHOUSE_HOST")
    clickhouse_port: int = Field(8123, env="CLICKHOUSE_PORT")
    clickhouse_user: str = Field("default", env="CLICKHOUSE_USER")
    clickhouse_password: str = Field("", env="CLICKHOUSE_PASSWORD")
    clickhouse_db: str = Field("default", env="CLICKHOUSE_DB")

    # HTTP
    http_port: int = Field(8000, env="HTTP_PORT")

    class Config:
        env_file = ".env"

settings = Settings()
