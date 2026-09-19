# System Metrics

Components required to collect system metrics from BOSH-deployed vms.

If you have any questions, or want to get attention for a PR or issue please reach out on the [#logging-and-metrics channel in the cloudfoundry slack](https://cloudfoundry.slack.com/archives/CUW93AF3M)

![system-metrics-architecture]


### System Metrics Agent
A standalone agent to provide VM system metrics via a prometheus-scrapable endpoint. A list of metrics
is available in the [docs][system-metrics-agent]

#### Metric Scraper
A central component for scraping `system-metrics-agents` and forwarding the metrics to the firehose. Metric Scraper
attempts to scrape the configured port across all vms deployed to the director. If present, this job can be configured to
communicate with the Leadership Election Job so duplicate scrapes are avoided in an HA environment. The system metrics scraper
can be found in the [system-metrics-scraper-release][system-metrics-scraper]

### Leadership Election
A job intended to be run alongside the System Metric Scraper to allow for multiple scrapers to exist while only one is 
scraping. 

[system-metrics-scraper]: https://github.com/cloudfoundry/system-metrics-scraper-release
[system-metrics-agent]: docs/system-metrics-agent.md
[system-metrics-architecture]: docs/system-metrics-architecture.png

+++++
