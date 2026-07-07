# Clock Drift Metrics

When the `clock_drift_enabled` BOSH property is set to `true`, the `system-metrics-agent` emits a lean set of metric families designed to satisfy PCI-DSS 10.6 audit requirements while providing essential operational visibility.

These metrics are collected by executing `chronyc tracking` on the host VM.

## PCI-DSS Audit Evidence (The "Must Haves")

These metrics provide the core evidence required by auditors to verify that critical systems are synchronized to a single, consistent time source.

*   **`clock_drift_last_offset_seconds` (Gauge)**: The raw offset from the last chrony poll. Proves the clock is actively synchronized within acceptable bounds.
*   **`clock_drift_leap_status` (Gauge)**: One-hot encoded status (`normal`, `insert`, `delete`, `unsync`, `unknown`). A value of `1.0` for `status="unsync"` is an immediate compliance failure and should trigger a Sev1 alert.
*   **`clock_drift_reference_info` (Gauge)**: Emits `1.0` with labels for `reference_id` (decoded to IP or ASCII) and `reference_host`. Proves all systems are acquiring time from the approved corporate time source.
*   **`clock_drift_collection_errors` (Gauge)**: Monotonically increasing counter of chronyc execution/parsing failures.

## Operational Best Practices (The "Highly Recommended")

These metrics provide additional context for platform engineers to diagnose time synchronization issues.

*   **`clock_drift_system_time_offset_seconds` (Gauge)**: The smoothed system time offset. Often better for alerting than the raw `last_offset` to avoid false positives.
*   **`clock_drift_stratum` (Gauge)**: The NTP stratum of the reference clock. An excellent at-a-glance health indicator for the foundation's time hierarchy.
*   **`clock_drift_frequency_ppm` (Gauge)**: The natural drift rate of the hardware clock. Massive spikes indicate underlying IaaS/hypervisor hardware issues.
*   **`clock_drift_root_delay_seconds` (Gauge)**: Network latency to the root time source. Useful for diagnosing UDP throttling or routing issues.

*(Note: Deeply technical chrony-internal smoothing metrics like RMS offset, skew, and dispersion have been explicitly excluded to minimize Prometheus cardinality and dashboard noise.)*
