# System Metrics
===================================================

Components required to collect system metrics from BOSH-deployed vms.
![system-metrics-architecture]

### System Metrics Agent
A standalone agent to provide VM system metrics via a prometheus-scrapable endpoint. A list of metrics
is available in the [docs][system-metrics-agent]. Can be scraped by the prom_scraper in loggregator-agent-release.
