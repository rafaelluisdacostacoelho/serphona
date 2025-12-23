from prometheus_client import Counter, Histogram, Gauge

M_CONSUMED = Counter("voc_events_consumed_total", "Total events consumed")
M_PROCESSED = Counter("voc_events_processed_total", "Total events successfully processed")
M_PROCESS_LATENCY = Histogram("voc_events_batch_process_seconds", "Batch processing time")
G_KAFKA_LAG = Gauge("voc_kafka_lag", "Kafka consumer lag (approx)")
