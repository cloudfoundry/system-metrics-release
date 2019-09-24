# System Metrics
[![slack.cloudfoundry.org][slack-badge]][loggregator-slack]
[![CI Badge][ci-badge]][ci-pipeline]
===================================================

Components required to collect system metrics from BOSH-deployed vms.
The architecture is described in this [diagram][system-metrics-architecture]

### System Metrics Agent
A standalone agent to provide VM system metrics via a prometheus-scrapable endpoint. A list of metrics
is available in the [docs][system-metrics-agent]

#### Metric Scraper
A central component for scraping `system-metrics-agents` and forwarding the metrics to the firehose. Metric Scraper
attempts to scrape the configured port across all vms deployed to the director. If present, this job can be configured to
communicate with the Leadership Election Job so duplicate scrapes are avoided in an HA environment.

### Leadership Election
A job intended to be run alongside the System Metric Scraper to allow for multiple scrapers to exist while only one is 
scraping. 

[system-metrics-agent]: docs/system-metrics-agent.md
[system-metrics-architecture]: docs/system-metrics-architecture.png
[slack-badge]:         https://slack.cloudfoundry.org/badge.svg
[loggregator-slack]:   https://cloudfoundry.slack.com/archives/loggregator
[ci-badge]:            https://loggregator.ci.cf-app.com/api/v1/pipelines/loggregator/jobs/loggregator-agent-tests/badge
[ci-pipeline]:         https://loggregator.ci.cf-app.com/teams/main/pipelines/loggregator
[loggregator-tracker]: https://www.pivotaltracker.com/n/projects/993188
