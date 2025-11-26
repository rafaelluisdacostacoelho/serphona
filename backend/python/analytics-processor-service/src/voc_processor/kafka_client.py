from aiokafka import AIOKafkaConsumer

def make_consumer(bootstrap, group_id, topics):
    return AIOKafkaConsumer(
        *topics,
        bootstrap_servers=bootstrap,
        group_id=group_id,
        enable_auto_commit=False,
        auto_offset_reset="earliest",
    )
