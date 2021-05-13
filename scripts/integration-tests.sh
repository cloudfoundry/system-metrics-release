#!/bin/bash

set -eo pipefail

time=$(date +%s%N)
cf install-plugin "log-cache" -f
sleep 120
cf tail system_metrics_agent -n 1000 --start-time="${time}" | grep system_cpu_core_idle


