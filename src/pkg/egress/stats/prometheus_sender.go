package stats

import (
	"math"

	"code.cloudfoundry.org/system-metrics-release/src/pkg/collector"
	"code.cloudfoundry.org/system-metrics-release/src/pkg/collector/clockdrift"
)

type Gauge interface {
	Set(float64)
}

type GaugeRegistry interface {
	Get(gaugeName, origin, unit string, tags map[string]string) Gauge
}

type PromSender struct {
	registry       GaugeRegistry
	origin         string
	labels         map[string]string
	limitedMetrics bool
}

func NewPromSender(registry GaugeRegistry, origin string, limitedMetrics bool, labels map[string]string) *PromSender {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["origin"] = origin
	return &PromSender{
		registry:       registry,
		origin:         origin,
		labels:         labels,
		limitedMetrics: limitedMetrics,
	}
}

func (p PromSender) Send(stats collector.SystemStat) {
	p.setSystemStats(stats)
	p.setSystemDiskGauges(stats)
	p.setEphemeralDiskGauges(stats)
	p.setPersistentDiskGauges(stats)
	// Clock drift metrics are NOT gated by limitedMetrics: PCI-DSS 10.4.1
	// audit evidence must be available regardless of cost-mode opt-outs, and
	// the trimmed lean set is small enough that cost is not a concern.
	p.setClockDriftGauges(stats)

	for _, network := range stats.Networks {
		p.setNetworkGauges(network)
	}
}

func clone(source map[string]string) map[string]string {
	copy := make(map[string]string)
	for k, v := range source {
		copy[k] = v
	}
	return copy
}

func (p PromSender) setSystemStats(stats collector.SystemStat) {
	labels := p.labels

	gauge := p.registry.Get("system_cpu_sys", p.origin, "Percent", labels)
	gauge.Set(float64(stats.System))

	gauge = p.registry.Get("system_cpu_wait", p.origin, "Percent", labels)
	gauge.Set(float64(stats.Wait))

	if !p.limitedMetrics {
		gauge = p.registry.Get("system_cpu_idle", p.origin, "Percent", labels)
		gauge.Set(float64(stats.Idle))
	}

	gauge = p.registry.Get("system_cpu_physical_core_count", p.origin, "Cores", labels)
	gauge.Set(float64(stats.CPUPhysicalCoreCount))

	gauge = p.registry.Get("system_cpu_threads_per_core", p.origin, "Threads", labels)
	gauge.Set(float64(stats.CPUThreadsPerCore))

	gauge = p.registry.Get("system_cpu_user", p.origin, "Percent", labels)
	gauge.Set(float64(stats.User))

	if !p.limitedMetrics {
		for _, perCoreStat := range stats.CPUCoreStats {
			perCoreLabels := clone(p.labels)
			perCoreLabels["cpu_name"] = perCoreStat.CPU

			gauge = p.registry.Get("system_cpu_core_sys", p.origin, "Percent", perCoreLabels)
			gauge.Set(float64(perCoreStat.System))

			gauge = p.registry.Get("system_cpu_core_wait", p.origin, "Percent", perCoreLabels)
			gauge.Set(float64(perCoreStat.Wait))

			gauge = p.registry.Get("system_cpu_core_idle", p.origin, "Percent", perCoreLabels)
			gauge.Set(float64(perCoreStat.Idle))

			gauge = p.registry.Get("system_cpu_core_user", p.origin, "Percent", perCoreLabels)
			gauge.Set(float64(perCoreStat.User))
		}
	}

	gauge = p.registry.Get("system_mem_kb", p.origin, "KiB", labels)
	gauge.Set(float64(stats.MemKB))

	gauge = p.registry.Get("system_mem_percent", p.origin, "Percent", labels)
	gauge.Set(float64(stats.MemPercent))

	gauge = p.registry.Get("system_swap_kb", p.origin, "KiB", labels)
	gauge.Set(float64(stats.SwapKB))

	gauge = p.registry.Get("system_swap_percent", p.origin, "Percent", labels)
	gauge.Set(float64(stats.SwapPercent))

	gauge = p.registry.Get("system_load_1m", p.origin, "Load", labels)
	gauge.Set(float64(stats.Load1M))

	if !p.limitedMetrics {
		gauge = p.registry.Get("system_load_5m", p.origin, "Load", labels)
		gauge.Set(float64(stats.Load5M))

		gauge = p.registry.Get("system_load_15m", p.origin, "Load", labels)
		gauge.Set(float64(stats.Load15M))

		gauge = p.registry.Get("system_network_ip_forwarding", p.origin, "", labels)
		gauge.Set(float64(stats.ProtoCounters.IPForwarding))

		gauge = p.registry.Get("system_network_udp_no_ports", p.origin, "", labels)
		gauge.Set(float64(stats.ProtoCounters.UDPNoPorts))

		gauge = p.registry.Get("system_network_udp_in_errors", p.origin, "", labels)
		gauge.Set(float64(stats.ProtoCounters.UDPInErrors))

		gauge = p.registry.Get("system_network_udp_lite_in_errors", p.origin, "", labels)
		gauge.Set(float64(stats.ProtoCounters.UDPLiteInErrors))

		gauge = p.registry.Get("system_network_tcp_active_opens", p.origin, "", labels)
		gauge.Set(float64(stats.ProtoCounters.TCPActiveOpens))

		gauge = p.registry.Get("system_network_tcp_curr_estab", p.origin, "", labels)
		gauge.Set(float64(stats.ProtoCounters.TCPCurrEstab))

		gauge = p.registry.Get("system_network_tcp_retrans_segs", p.origin, "", labels)
		gauge.Set(float64(stats.ProtoCounters.TCPRetransSegs))
	}
	if stats.Health.Present {
		var healthValue float64
		if stats.Health.Healthy {
			healthValue = 1.0
		}

		gauge = p.registry.Get("system_healthy", p.origin, "", labels)
		gauge.Set(healthValue)
	}
}

func (p PromSender) setNetworkGauges(network collector.NetworkStat) {
	if p.limitedMetrics {
		return
	}

	labels := clone(p.labels)
	labels["network_interface"] = network.Name

	gauge := p.registry.Get("system_network_bytes_sent", p.origin, "Bytes", labels)
	gauge.Set(float64(network.BytesSent))

	gauge = p.registry.Get("system_network_bytes_received", p.origin, "Bytes", labels)
	gauge.Set(float64(network.BytesReceived))

	gauge = p.registry.Get("system_network_packets_sent", p.origin, "Packets", labels)
	gauge.Set(float64(network.PacketsSent))

	gauge = p.registry.Get("system_network_packets_received", p.origin, "Packets", labels)
	gauge.Set(float64(network.PacketsReceived))

	gauge = p.registry.Get("system_network_error_in", p.origin, "Frames", labels)
	gauge.Set(float64(network.ErrIn))

	gauge = p.registry.Get("system_network_error_out", p.origin, "Frames", labels)
	gauge.Set(float64(network.ErrOut))

	gauge = p.registry.Get("system_network_drop_in", p.origin, "Packets", labels)
	gauge.Set(float64(network.DropIn))

	gauge = p.registry.Get("system_network_drop_out", p.origin, "Packets", labels)
	gauge.Set(float64(network.DropOut))
}

func (p PromSender) setSystemDiskGauges(stats collector.SystemStat) {
	if !stats.SystemDisk.Present {
		return
	}
	labels := p.labels

	gauge := p.registry.Get("system_disk_system_percent", p.origin, "Percent", labels)
	gauge.Set(float64(stats.SystemDisk.Percent))

	gauge = p.registry.Get("system_disk_system_inode_percent", p.origin, "Percent", labels)
	gauge.Set(float64(stats.SystemDisk.InodePercent))

	if !p.limitedMetrics {
		gauge = p.registry.Get("system_disk_system_read_bytes", p.origin, "Bytes", labels)
		gauge.Set(float64(stats.SystemDisk.ReadBytes))

		gauge = p.registry.Get("system_disk_system_write_bytes", p.origin, "Bytes", labels)
		gauge.Set(float64(stats.SystemDisk.WriteBytes))

		gauge = p.registry.Get("system_disk_system_read_time", p.origin, "ms", labels)
		gauge.Set(float64(stats.SystemDisk.ReadTime))

		gauge = p.registry.Get("system_disk_system_write_time", p.origin, "ms", labels)
		gauge.Set(float64(stats.SystemDisk.WriteTime))

		gauge = p.registry.Get("system_disk_system_io_time", p.origin, "ms", labels)
		gauge.Set(float64(stats.SystemDisk.IOTime))
	}
}

func (p PromSender) setEphemeralDiskGauges(stats collector.SystemStat) {
	if !stats.EphemeralDisk.Present {
		return
	}
	labels := p.labels

	gauge := p.registry.Get("system_disk_ephemeral_percent", p.origin, "Percent", labels)
	gauge.Set(float64(stats.EphemeralDisk.Percent))

	gauge = p.registry.Get("system_disk_ephemeral_inode_percent", p.origin, "Percent", labels)
	gauge.Set(float64(stats.EphemeralDisk.InodePercent))

	if !p.limitedMetrics {
		gauge = p.registry.Get("system_disk_ephemeral_read_bytes", p.origin, "Bytes", labels)
		gauge.Set(float64(stats.EphemeralDisk.ReadBytes))

		gauge = p.registry.Get("system_disk_ephemeral_write_bytes", p.origin, "Bytes", labels)
		gauge.Set(float64(stats.EphemeralDisk.WriteBytes))

		gauge = p.registry.Get("system_disk_ephemeral_read_time", p.origin, "ms", labels)
		gauge.Set(float64(stats.EphemeralDisk.ReadTime))

		gauge = p.registry.Get("system_disk_ephemeral_write_time", p.origin, "ms", labels)
		gauge.Set(float64(stats.EphemeralDisk.WriteTime))

		gauge = p.registry.Get("system_disk_ephemeral_io_time", p.origin, "ms", labels)
		gauge.Set(float64(stats.EphemeralDisk.IOTime))
	}
}

func (p PromSender) setPersistentDiskGauges(stats collector.SystemStat) {
	if !stats.PersistentDisk.Present {
		return
	}
	labels := p.labels

	gauge := p.registry.Get("system_disk_persistent_percent", p.origin, "Percent", labels)
	gauge.Set(float64(stats.PersistentDisk.Percent))

	gauge = p.registry.Get("system_disk_persistent_inode_percent", p.origin, "Percent", labels)
	gauge.Set(float64(stats.PersistentDisk.InodePercent))

	if !p.limitedMetrics {
		gauge = p.registry.Get("system_disk_persistent_read_bytes", p.origin, "Bytes", labels)
		gauge.Set(float64(stats.PersistentDisk.ReadBytes))

		gauge = p.registry.Get("system_disk_persistent_write_bytes", p.origin, "Bytes", labels)
		gauge.Set(float64(stats.PersistentDisk.WriteBytes))

		gauge = p.registry.Get("system_disk_persistent_read_time", p.origin, "ms", labels)
		gauge.Set(float64(stats.PersistentDisk.ReadTime))

		gauge = p.registry.Get("system_disk_persistent_write_time", p.origin, "ms", labels)
		gauge.Set(float64(stats.PersistentDisk.WriteTime))

		gauge = p.registry.Get("system_disk_persistent_io_time", p.origin, "ms", labels)
		gauge.Set(float64(stats.PersistentDisk.IOTime))
	}
}

// setClockDriftGauges emits PCI-DSS 10.6 audit-evidence metrics for the
// local clock's NTP-sync state.
//
// Design notes:
//   - The reference id/host pair is exposed once via clock_drift_reference_info
//     rather than as labels on every numeric gauge. Promoting them to per-gauge
//     labels would multiply Prometheus cardinality by N peers per chrony
//     reselection -- a slow-burn memory leak in PromRegistry that long-lived
//     VMs cannot recover from. The reference info gauge itself is the only
//     surface where peer rotation widens the series set, and it is bounded
//     in practice (a typical VM sees <10 peers over its lifetime).
//   - LeapStatus is one gauge with a status label, following the
//     kube-state-metrics convention. This lets dashboards distinguish
//     LeapUnknown (parse failure) from LeapNotSynchronised, which the previous
//     four-boolean design could not.
//   - Numeric fields use math.NaN() (or nil pointers) as a sentinel for
//     "the underlying tool emitted a value we could not parse". The gauge is
//     deliberately NOT set in that case so a parser regression cannot manifest
//     as a misleading zero -- which an auditor would read as "perfect clock
//     sync" and miss the underlying breakage.
//   - clock_drift_collection_errors is emitted whenever clock drift
//     collection is enabled (even if the latest cycle succeeded), so
//     operators can alert on rate(...) > 0 without the metric flickering in
//     and out of existence. The metric is named without the _total suffix
//     because Prometheus reserves _total for Counter; we expose a Gauge that
//     happens to monotonically increase in-process, and rate() works on it
//     identically.
func (p PromSender) setClockDriftGauges(stats collector.SystemStat) {
	if !stats.ClockDriftEnabled {
		return
	}

	labels := p.labels

	errs := p.registry.Get("clock_drift_collection_errors", p.origin, "", labels)
	errs.Set(float64(stats.ClockDriftErrorsTotal))

	if stats.ClockDrift == nil {
		return
	}

	p.setClockDriftReferenceInfo(stats.ClockDrift)
	p.setClockDriftLeapStatus(stats.ClockDrift)
	p.setClockDriftNumericGauges(stats.ClockDrift)
}

func (p PromSender) setClockDriftReferenceInfo(d *clockdrift.TimeSyncData) {
	infoLabels := clone(p.labels)
	infoLabels["reference_id"] = nonEmpty(d.ReferenceID, "unknown")
	infoLabels["reference_host"] = nonEmpty(d.ReferenceHost, "unknown")
	infoLabels["source"] = clockdrift.BackendChrony

	gauge := p.registry.Get("clock_drift_reference_info", p.origin, "", infoLabels)
	gauge.Set(1.0)
}

func (p PromSender) setClockDriftLeapStatus(d *clockdrift.TimeSyncData) {
	for _, status := range clockdrift.LeapStatusValues {
		labels := clone(p.labels)
		labels["status"] = leapStatusLabel(status)

		val := 0.0
		if d.LeapStatus == status {
			val = 1.0
		}
		gauge := p.registry.Get("clock_drift_leap_status", p.origin, "", labels)
		gauge.Set(val)
	}
}

// setClockDriftNumericGauges emits the lean PCI-DSS-plus-operations set:
// system_time_offset_seconds and last_offset_seconds for compliance evidence,
// frequency_ppm and root_delay_seconds for IaaS/network diagnostics, and
// stratum as the at-a-glance health indicator. The chrony-internal smoothing
// and error-bound metrics (rms_offset, residual_freq_ppm, skew_ppm,
// root_dispersion, update_interval, ref_time_unix) are deliberately
// commented out: they have no operational or audit value at the foundation
// scope, and uncommenting any line below is sufficient to bring them back.
// The clockdrift parser still populates every TimeSyncData field unchanged.
func (p PromSender) setClockDriftNumericGauges(d *clockdrift.TimeSyncData) {
	labels := p.labels

	setIfFinite := func(name, unit string, value float64) {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return
		}
		gauge := p.registry.Get(name, p.origin, unit, labels)
		gauge.Set(value)
	}

	setIfFinite("clock_drift_system_time_offset_seconds", "Seconds", d.SystemTimeOffsetSec)
	setIfFinite("clock_drift_last_offset_seconds", "Seconds", d.LastOffsetSec)
	setIfFinite("clock_drift_frequency_ppm", "ppm", d.FrequencyPPM)
	setIfFinite("clock_drift_root_delay_seconds", "Seconds", d.RootDelaySec)

	// Trimmed -- see helper doc above.
	// setIfFinite("clock_drift_rms_offset_seconds", "Seconds", d.RMSOffsetSec)
	// setIfFinite("clock_drift_residual_freq_ppm", "ppm", d.ResidualFreqPPM)
	// setIfFinite("clock_drift_skew_ppm", "ppm", d.SkewPPM)
	// setIfFinite("clock_drift_root_dispersion_seconds", "Seconds", d.RootDispersionSec)
	// setIfFinite("clock_drift_update_interval_seconds", "Seconds", d.UpdateIntervalSec)

	if d.Stratum != nil {
		gauge := p.registry.Get("clock_drift_stratum", p.origin, "Stratum", labels)
		gauge.Set(float64(*d.Stratum))
	}

	// Trimmed -- see helper doc above.
	// if d.RefTimeUnixSec != nil {
	// 	gauge := p.registry.Get("clock_drift_ref_time_unix_seconds", p.origin, "Seconds", labels)
	// 	gauge.Set(*d.RefTimeUnixSec)
	// }
}

func nonEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// leapStatusLabel returns the lowercased dashboard-friendly label value for
// a chrony LeapStatus. Stable strings here matter: dashboards key off of
// them, so we centralize the mapping rather than spreading raw enum-to-label
// conversions across the file.
func leapStatusLabel(s clockdrift.LeapStatus) string {
	switch s {
	case clockdrift.LeapNormal:
		return "normal"
	case clockdrift.LeapInsertSecond:
		return "insert"
	case clockdrift.LeapDeleteSecond:
		return "delete"
	case clockdrift.LeapNotSynchronised:
		return "unsync"
	default:
		return "unknown"
	}
}
