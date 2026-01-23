# Usage Publisher Alerts

Prometheus alerting rules for Tools Gateway usage publishing are defined in `usage-publisher-rules.yaml`.

## How to deploy
- Include the rule file in your Prometheus/Alertmanager stack (Helm values or rules mount).
- Ensure Prometheus is scraping Tools Gateway metrics endpoint `/metrics`.
- Recommended scrape interval: 15s–30s; rule group interval is 30s.

## Alert set
- `UsagePublisherRetrySpike`: retryable errors >20% of publishes over 5m (per sink).
- `UsagePublisherBreakerOpen`: breaker open for >2m (per sink/target).
- `UsagePublisherRateLimited`: sustained 429s >0.1/s over 5m (per sink/target).
