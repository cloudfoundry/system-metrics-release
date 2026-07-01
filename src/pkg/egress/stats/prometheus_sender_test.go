package stats_test

import (
	"math"

	"code.cloudfoundry.org/system-metrics-release/src/pkg/collector"
	"code.cloudfoundry.org/system-metrics-release/src/pkg/collector/clockdrift"
	"code.cloudfoundry.org/system-metrics-release/src/pkg/egress/stats"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// gaugeKey reproduces the simple stub-registry key for gauges that have no
// extra label dimensions beyond the default tag set. Used by clock-drift
// numeric-gauge assertions.
func gaugeKey(name, origin, unit string) string {
	return name + origin + unit
}

// leapLabel mirrors the sender's internal leap-status-to-label conversion.
// Tests intentionally hardcode the strings to catch unexpected renames in
// the production code -- if a label changes, dashboards break, so a failing
// test is the right signal.
func leapLabel(s clockdrift.LeapStatus) string {
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

var _ = Describe("Prometheus Sender", func() {
	var (
		sender   *stats.PromSender
		registry *stubRegistry
		labels   map[string]string
	)

	BeforeEach(func() {
		registry = newStubRegistry()
		labels = map[string]string{
			"source_id":  "test-origin",
			"deployment": "test-deployment",
			"job":        "test-job",
			"index":      "test-index",
			"ip":         "test-ip",
		}

		sender = stats.NewPromSender(registry, "test-origin", false, labels)
	})

	It("gets the correct number of metrics from the registry", func() {
		sender.Send(defaultInput)

		// 50 = system stats + disk + network proto counters + health.
		// Clock drift metrics are not counted here because defaultInput
		// has ClockDriftEnabled=false (the zero value), which short-circuits
		// the entire setClockDriftGauges path before any gauge is created.
		Expect(registry.gaugeCount).To(Equal(50))
	})

	It("does not panic with no default labels", func() {
		stats.NewPromSender(registry, "test-origin", false, nil)
	})

	DescribeTable("default tags", func(tag, value string) {
		sender.Send(defaultInput)

		gauge := registry.gauges["system_mem_kbtest-originKiB"]

		Expect(gauge.tags[tag]).To(Equal(value))
	},
		Entry("origin", "origin", "test-origin"),
		Entry("source_id", "source_id", "test-origin"),
		Entry("deployment", "deployment", "test-deployment"),
		Entry("job", "job", "test-job"),
		Entry("index", "index", "test-index"),
		Entry("ip", "ip", "test-ip"),
	)

	DescribeTable("default metrics", func(name string, tags map[string]string, unit string, value float64) {
		sender.Send(defaultInput)

		cpuName := ""
		if cpuTag, ok := tags["cpu_name"]; ok {
			cpuName = cpuTag
		}

		keyName := name + tags["origin"] + unit + cpuName
		gauge := registry.gauges[keyName]

		Expect(gauge).NotTo(BeNil())
		Expect(gauge.value).To(BeNumerically("==", value))

		for k, v := range tags {
			Expect(gauge.tags[k]).To(Equal(v))
		}
	},
		Entry("system_mem_kb", "system_mem_kb", map[string]string{"origin": "test-origin"}, "KiB", 1025.0),
		Entry("system_mem_percent", "system_mem_percent", map[string]string{"origin": "test-origin"}, "Percent", 10.01),
		Entry("system_swap_kb", "system_swap_kb", map[string]string{"origin": "test-origin"}, "KiB", 2049.0),
		Entry("system_swap_percent", "system_swap_percent", map[string]string{"origin": "test-origin"}, "Percent", 20.01),
		Entry("system_load_1m", "system_load_1m", map[string]string{"origin": "test-origin"}, "Load", 1.1),
		Entry("system_load_5m", "system_load_5m", map[string]string{"origin": "test-origin"}, "Load", 5.5),
		Entry("system_load_15m", "system_load_15m", map[string]string{"origin": "test-origin"}, "Load", 15.15),
		Entry("system_cpu_user", "system_cpu_user", map[string]string{"origin": "test-origin"}, "Percent", 25.25),
		Entry("system_cpu_sys", "system_cpu_sys", map[string]string{"origin": "test-origin"}, "Percent", 52.52),
		Entry("system_cpu_idle", "system_cpu_idle", map[string]string{"origin": "test-origin"}, "Percent", 10.10),
		Entry("system_cpu_wait", "system_cpu_wait", map[string]string{"origin": "test-origin"}, "Percent", 22.22),
		Entry("system_cpu_physical_core_count", "system_cpu_physical_core_count", map[string]string{"origin": "test-origin"}, "Cores", 1.0),
		Entry("system_cpu_threads_per_core", "system_cpu_threads_per_core", map[string]string{"origin": "test-origin"}, "Threads", 2.0),
		Entry("system cpu core 1 user", "system_cpu_core_user", map[string]string{"origin": "test-origin", "cpu_name": "cpu1"}, "Percent", 25.25),
		Entry("system cpu core 1 sys", "system_cpu_core_sys", map[string]string{"origin": "test-origin", "cpu_name": "cpu1"}, "Percent", 52.52),
		Entry("system cpu core 1 idle", "system_cpu_core_idle", map[string]string{"origin": "test-origin", "cpu_name": "cpu1"}, "Percent", 10.10),
		Entry("system cpu core 1 wait", "system_cpu_core_wait", map[string]string{"origin": "test-origin", "cpu_name": "cpu1"}, "Percent", 22.22),
		Entry("system cpu core 2 user", "system_cpu_core_user", map[string]string{"origin": "test-origin", "cpu_name": "cpu2"}, "Percent", 25.25),
		Entry("system cpu core 2 sys", "system_cpu_core_sys", map[string]string{"origin": "test-origin", "cpu_name": "cpu2"}, "Percent", 52.52),
		Entry("system cpu core 2 idle", "system_cpu_core_idle", map[string]string{"origin": "test-origin", "cpu_name": "cpu2"}, "Percent", 10.10),
		Entry("system cpu core 2 wait", "system_cpu_core_wait", map[string]string{"origin": "test-origin", "cpu_name": "cpu2"}, "Percent", 22.22),
		Entry("system_disk_system_percent", "system_disk_system_percent", map[string]string{"origin": "test-origin"}, "Percent", 35.0),
		Entry("system_disk_system_inode_percent", "system_disk_system_inode_percent", map[string]string{"origin": "test-origin"}, "Percent", 45.0),
		Entry("system_disk_system_read_bytes", "system_disk_system_read_bytes", map[string]string{"origin": "test-origin"}, "Bytes", 10.0),
		Entry("system_disk_system_write_bytes", "system_disk_system_write_bytes", map[string]string{"origin": "test-origin"}, "Bytes", 20.0),
		Entry("system_disk_system_read_time", "system_disk_system_read_time", map[string]string{"origin": "test-origin"}, "ms", 30.0),
		Entry("system_disk_system_write_time", "system_disk_system_write_time", map[string]string{"origin": "test-origin"}, "ms", 40.0),
		Entry("system_disk_system_io_time", "system_disk_system_io_time", map[string]string{"origin": "test-origin"}, "ms", 50.0),
		Entry("system_disk_ephemeral_percent", "system_disk_ephemeral_percent", map[string]string{"origin": "test-origin"}, "Percent", 55.0),
		Entry("system_disk_ephemeral_inode_percent", "system_disk_ephemeral_inode_percent", map[string]string{"origin": "test-origin"}, "Percent", 65.0),
		Entry("system_disk_ephemeral_read_bytes", "system_disk_ephemeral_read_bytes", map[string]string{"origin": "test-origin"}, "Bytes", 100.0),
		Entry("system_disk_ephemeral_write_bytes", "system_disk_ephemeral_write_bytes", map[string]string{"origin": "test-origin"}, "Bytes", 200.0),
		Entry("system_disk_ephemeral_read_time", "system_disk_ephemeral_read_time", map[string]string{"origin": "test-origin"}, "ms", 300.0),
		Entry("system_disk_ephemeral_write_time", "system_disk_ephemeral_write_time", map[string]string{"origin": "test-origin"}, "ms", 400.0),
		Entry("system_disk_ephemeral_io_time", "system_disk_ephemeral_io_time", map[string]string{"origin": "test-origin"}, "ms", 500.0),
		Entry("system_disk_persistent_percent", "system_disk_persistent_percent", map[string]string{"origin": "test-origin"}, "Percent", 75.0),
		Entry("system_disk_persistent_inode_percent", "system_disk_persistent_inode_percent", map[string]string{"origin": "test-origin"}, "Percent", 85.0),
		Entry("system_disk_persistent_read_bytes", "system_disk_persistent_read_bytes", map[string]string{"origin": "test-origin"}, "Bytes", 1000.0),
		Entry("system_disk_persistent_write_bytes", "system_disk_persistent_write_bytes", map[string]string{"origin": "test-origin"}, "Bytes", 2000.0),
		Entry("system_disk_persistent_read_time", "system_disk_persistent_read_time", map[string]string{"origin": "test-origin"}, "ms", 3000.0),
		Entry("system_disk_persistent_write_time", "system_disk_persistent_write_time", map[string]string{"origin": "test-origin"}, "ms", 4000.0),
		Entry("system_disk_persistent_io_time", "system_disk_persistent_io_time", map[string]string{"origin": "test-origin"}, "ms", 5000.0),
		Entry("system_healthy", "system_healthy", map[string]string{"origin": "test-origin"}, "", 1.0),
	)

	DescribeTable("limited metrics", func(name string, tags map[string]string, unit string, exists bool) {
		sender = stats.NewPromSender(registry, "test-origin", true, labels)
		sender.Send(defaultInput)

		cpuName := ""
		if cpuTag, ok := tags["cpu_name"]; ok {
			cpuName = cpuTag
		}

		keyName := name + tags["origin"] + unit + cpuName
		gauge := registry.gauges[keyName]

		if exists {
			Expect(gauge).NotTo(BeNil())
			Expect(gauge.value).ToNot(BeZero())
			for k, v := range tags {
				Expect(gauge.tags[k]).To(Equal(v))
			}
		} else {
			Expect(gauge).To(BeNil())
		}
	},
		Entry("system_mem_kb", "system_mem_kb", map[string]string{"origin": "test-origin"}, "KiB", true),
		Entry("system_mem_percent", "system_mem_percent", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system_swap_kb", "system_swap_kb", map[string]string{"origin": "test-origin"}, "KiB", true),
		Entry("system_swap_percent", "system_swap_percent", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system_load_1m", "system_load_1m", map[string]string{"origin": "test-origin"}, "Load", true),
		Entry("system_load_5m", "system_load_5m", map[string]string{"origin": "test-origin"}, "Load", false),
		Entry("system_load_15m", "system_load_15m", map[string]string{"origin": "test-origin"}, "Load", false),
		Entry("system_cpu_user", "system_cpu_user", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system_cpu_sys", "system_cpu_sys", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system_cpu_idle", "system_cpu_idle", map[string]string{"origin": "test-origin"}, "Percent", false),
		Entry("system_cpu_wait", "system_cpu_wait", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system cpu core 1 user", "system_cpu_core_user", map[string]string{"origin": "test-origin", "cpu_name": "cpu1"}, "Percent", false),
		Entry("system cpu core 1 sys", "system_cpu_core_sys", map[string]string{"origin": "test-origin", "cpu_name": "cpu1"}, "Percent", false),
		Entry("system cpu core 1 idle", "system_cpu_core_idle", map[string]string{"origin": "test-origin", "cpu_name": "cpu1"}, "Percent", false),
		Entry("system cpu core 1 wait", "system_cpu_core_wait", map[string]string{"origin": "test-origin", "cpu_name": "cpu1"}, "Percent", false),
		Entry("system cpu core 2 user", "system_cpu_core_user", map[string]string{"origin": "test-origin", "cpu_name": "cpu2"}, "Percent", false),
		Entry("system cpu core 2 sys", "system_cpu_core_sys", map[string]string{"origin": "test-origin", "cpu_name": "cpu2"}, "Percent", false),
		Entry("system cpu core 2 idle", "system_cpu_core_idle", map[string]string{"origin": "test-origin", "cpu_name": "cpu2"}, "Percent", false),
		Entry("system cpu core 2 wait", "system_cpu_core_wait", map[string]string{"origin": "test-origin", "cpu_name": "cpu2"}, "Percent", false),
		Entry("system_disk_system_percent", "system_disk_system_percent", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system_disk_system_inode_percent", "system_disk_system_inode_percent", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system_disk_system_read_bytes", "system_disk_system_read_bytes", map[string]string{"origin": "test-origin"}, "Bytes", false),
		Entry("system_disk_system_write_bytes", "system_disk_system_write_bytes", map[string]string{"origin": "test-origin"}, "Bytes", false),
		Entry("system_disk_system_read_time", "system_disk_system_read_time", map[string]string{"origin": "test-origin"}, "ms", false),
		Entry("system_disk_system_write_time", "system_disk_system_write_time", map[string]string{"origin": "test-origin"}, "ms", false),
		Entry("system_disk_system_io_time", "system_disk_system_io_time", map[string]string{"origin": "test-origin"}, "ms", false),
		Entry("system_disk_ephemeral_percent", "system_disk_ephemeral_percent", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system_disk_ephemeral_inode_percent", "system_disk_ephemeral_inode_percent", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system_disk_ephemeral_read_bytes", "system_disk_ephemeral_read_bytes", map[string]string{"origin": "test-origin"}, "Bytes", false),
		Entry("system_disk_ephemeral_write_bytes", "system_disk_ephemeral_write_bytes", map[string]string{"origin": "test-origin"}, "Bytes", false),
		Entry("system_disk_ephemeral_read_time", "system_disk_ephemeral_read_time", map[string]string{"origin": "test-origin"}, "ms", false),
		Entry("system_disk_ephemeral_write_time", "system_disk_ephemeral_write_time", map[string]string{"origin": "test-origin"}, "ms", false),
		Entry("system_disk_ephemeral_io_time", "system_disk_ephemeral_io_time", map[string]string{"origin": "test-origin"}, "ms", false),
		Entry("system_disk_persistent_percent", "system_disk_persistent_percent", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system_disk_persistent_inode_percent", "system_disk_persistent_inode_percent", map[string]string{"origin": "test-origin"}, "Percent", true),
		Entry("system_disk_persistent_read_bytes", "system_disk_persistent_read_bytes", map[string]string{"origin": "test-origin"}, "Bytes", false),
		Entry("system_disk_persistent_write_bytes", "system_disk_persistent_write_bytes", map[string]string{"origin": "test-origin"}, "Bytes", false),
		Entry("system_disk_persistent_read_time", "system_disk_persistent_read_time", map[string]string{"origin": "test-origin"}, "ms", false),
		Entry("system_disk_persistent_write_time", "system_disk_persistent_write_time", map[string]string{"origin": "test-origin"}, "ms", false),
		Entry("system_disk_persistent_io_time", "system_disk_persistent_io_time", map[string]string{"origin": "test-origin"}, "ms", false),
		Entry("system_healthy", "system_healthy", map[string]string{"origin": "test-origin"}, "", true),
	)

	DescribeTable("network metrics with limited metrics", func(name, origin, unit, networkName string) {
		sender = stats.NewPromSender(registry, "test-origin", true, labels)
		sender.Send(networkInput)

		_, exists := registry.gauges[name+origin+unit+networkName]
		Expect(exists).To(BeFalse())
	},
		Entry("system_network_bytes_sent", "system_network_bytes_sent", "test-origin", "Bytes", "eth0"),
		Entry("system_network_bytes_received", "system_network_bytes_received", "test-origin", "Bytes", "eth0"),
		Entry("system_network_packets_sent", "system_network_packets_sent", "test-origin", "Packets", "eth0"),
		Entry("system_network_packets_received", "system_network_packets_received", "test-origin", "Packets", "eth0"),
		Entry("system_network_error_in", "system_network_error_in", "test-origin", "Frames", "eth0"),
		Entry("system_network_error_out", "system_network_error_out", "test-origin", "Frames", "eth0"),
		Entry("system_network_drop_in", "system_network_drop_in", "test-origin", "Packets", "eth0"),
		Entry("system_network_drop_out", "system_network_drop_out", "test-origin", "Packets", "eth0"),

		Entry("system_network_bytes_sent", "system_network_bytes_sent", "test-origin", "Bytes", "eth1"),
		Entry("system_network_bytes_received", "system_network_bytes_received", "test-origin", "Bytes", "eth1"),
		Entry("system_network_packets_sent", "system_network_packets_sent", "test-origin", "Packets", "eth1"),
		Entry("system_network_packets_received", "system_network_packets_received", "test-origin", "Packets", "eth1"),
		Entry("system_network_error_in", "system_network_error_in", "test-origin", "Frames", "eth1"),
		Entry("system_network_error_out", "system_network_error_out", "test-origin", "Frames", "eth1"),
		Entry("system_network_drop_in", "system_network_drop_in", "test-origin", "Packets", "eth1"),
		Entry("system_network_drop_out", "system_network_drop_out", "test-origin", "Packets", "eth1"),
	)

	DescribeTable("network metrics", func(name, origin, unit, networkName string, value float64) {
		sender.Send(networkInput)

		gauge, exists := registry.gauges[name+origin+unit+networkName]
		Expect(exists).To(BeTrue())

		Expect(gauge.value).To(BeNumerically("==", value))
		Expect(gauge.tags["network_interface"]).To(Or(Equal("eth0"), Equal("eth1")))
		Expect(gauge.tags["origin"]).To(Equal("test-origin"))
	},
		Entry("system_network_bytes_sent", "system_network_bytes_sent", "test-origin", "Bytes", "eth0", 1.0),
		Entry("system_network_bytes_received", "system_network_bytes_received", "test-origin", "Bytes", "eth0", 2.0),
		Entry("system_network_packets_sent", "system_network_packets_sent", "test-origin", "Packets", "eth0", 3.0),
		Entry("system_network_packets_received", "system_network_packets_received", "test-origin", "Packets", "eth0", 4.0),
		Entry("system_network_error_in", "system_network_error_in", "test-origin", "Frames", "eth0", 5.0),
		Entry("system_network_error_out", "system_network_error_out", "test-origin", "Frames", "eth0", 6.0),
		Entry("system_network_drop_in", "system_network_drop_in", "test-origin", "Packets", "eth0", 7.0),
		Entry("system_network_drop_out", "system_network_drop_out", "test-origin", "Packets", "eth0", 8.0),

		Entry("system_network_bytes_sent", "system_network_bytes_sent", "test-origin", "Bytes", "eth1", 10.0),
		Entry("system_network_bytes_received", "system_network_bytes_received", "test-origin", "Bytes", "eth1", 20.0),
		Entry("system_network_packets_sent", "system_network_packets_sent", "test-origin", "Packets", "eth1", 30.0),
		Entry("system_network_packets_received", "system_network_packets_received", "test-origin", "Packets", "eth1", 40.0),
		Entry("system_network_error_in", "system_network_error_in", "test-origin", "Frames", "eth1", 50.0),
		Entry("system_network_error_out", "system_network_error_out", "test-origin", "Frames", "eth1", 60.0),
		Entry("system_network_drop_in", "system_network_drop_in", "test-origin", "Packets", "eth1", 70.0),
		Entry("system_network_drop_out", "system_network_drop_out", "test-origin", "Packets", "eth1", 80.0),
	)

	Describe("clock drift metrics", func() {
		var (
			stratum4   = 4
			refTimeSec = 12345.0
			fullInput  collector.SystemStat
		)

		BeforeEach(func() {
			fullInput = collector.SystemStat{
				ClockDriftEnabled: true,
				ClockDrift: &clockdrift.TimeSyncData{
					ReferenceID:         "REF1",
					ReferenceHost:       "1.2.3.4",
					Stratum:             &stratum4,
					RefTimeUnixSec:      &refTimeSec,
					SystemTimeOffsetSec: -0.123,
					LastOffsetSec:       -0.000364,
					RMSOffsetSec:        0.000758,
					FrequencyPPM:        10.5,
					ResidualFreqPPM:     -0.003,
					SkewPPM:             0.146,
					RootDelaySec:        0.070813,
					RootDispersionSec:   0.011881,
					UpdateIntervalSec:   2073.3,
					LeapStatus:          clockdrift.LeapNormal,
				},
			}
		})

		DescribeTable("numeric gauges", func(name, unit string, value float64) {
			sender.Send(fullInput)

			gauge := registry.gauges[gaugeKey(name, "test-origin", unit)]
			Expect(gauge).NotTo(BeNil(), "gauge %s not found", name)
			Expect(gauge.value).To(BeNumerically("==", value))
			Expect(gauge.tags).NotTo(HaveKey("reference_id"), "numeric gauges must NOT carry reference_id (cardinality)")
			Expect(gauge.tags).NotTo(HaveKey("reference_host"), "numeric gauges must NOT carry reference_host (cardinality)")
			Expect(gauge.tags["origin"]).To(Equal("test-origin"))
		},
			Entry("system_time_offset_seconds", "clock_drift_system_time_offset_seconds", "Seconds", -0.123),
			Entry("last_offset_seconds", "clock_drift_last_offset_seconds", "Seconds", -0.000364),
			Entry("frequency_ppm", "clock_drift_frequency_ppm", "ppm", 10.5),
			Entry("root_delay_seconds", "clock_drift_root_delay_seconds", "Seconds", 0.070813),
			Entry("stratum", "clock_drift_stratum", "Stratum", 4.0),
			// Trimmed -- chrony-internal smoothing/error-bound metrics with
			// no operational or audit value. Uncomment to bring back; the
			// parser still populates these fields on TimeSyncData.
			// Entry("rms_offset_seconds", "clock_drift_rms_offset_seconds", "Seconds", 0.000758),
			// Entry("residual_freq_ppm", "clock_drift_residual_freq_ppm", "ppm", -0.003),
			// Entry("skew_ppm", "clock_drift_skew_ppm", "ppm", 0.146),
			// Entry("root_dispersion_seconds", "clock_drift_root_dispersion_seconds", "Seconds", 0.011881),
			// Entry("update_interval_seconds", "clock_drift_update_interval_seconds", "Seconds", 2073.3),
			// Entry("ref_time_unix_seconds", "clock_drift_ref_time_unix_seconds", "Seconds", 12345.0),
		)

		It("does NOT emit the trimmed chrony-internal smoothing metrics", func() {
			sender.Send(fullInput)

			trimmed := []string{
				"clock_drift_rms_offset_seconds",
				"clock_drift_residual_freq_ppm",
				"clock_drift_skew_ppm",
				"clock_drift_root_dispersion_seconds",
				"clock_drift_update_interval_seconds",
				"clock_drift_ref_time_unix_seconds",
			}
			for _, name := range trimmed {
				Expect(registry.findByName(name)).To(BeNil(),
					"%s should not be emitted (trimmed from lean PCI set)", name)
			}
		})

		It("emits a single reference_info gauge with reference labels", func() {
			sender.Send(fullInput)

			gauge := registry.findByNameAndLabel("clock_drift_reference_info", "reference_id", "REF1")
			Expect(gauge).NotTo(BeNil(), "clock_drift_reference_info gauge not found")
			Expect(gauge.value).To(BeNumerically("==", 1.0))
			Expect(gauge.tags["reference_id"]).To(Equal("REF1"))
			Expect(gauge.tags["reference_host"]).To(Equal("1.2.3.4"))
			Expect(gauge.tags["source"]).To(Equal(clockdrift.BackendChrony))
		})

		It("emits one reference_info gauge per unique peer (cardinality follows peer rotation only)", func() {
			second := 2
			input2 := fullInput
			input2.ClockDrift = &clockdrift.TimeSyncData{
				ReferenceID:   "REF2",
				ReferenceHost: "5.6.7.8",
				Stratum:       &second,
				LeapStatus:    clockdrift.LeapNormal,
			}

			sender.Send(fullInput)
			sender.Send(input2)

			Expect(registry.findByNameAndLabel("clock_drift_reference_info", "reference_id", "REF1")).NotTo(BeNil())
			Expect(registry.findByNameAndLabel("clock_drift_reference_info", "reference_id", "REF2")).NotTo(BeNil())
		})

		It("falls back to 'unknown' when reference labels are empty (avoids label-key flapping)", func() {
			zero := 0
			input := collector.SystemStat{
				ClockDriftEnabled: true,
				ClockDrift: &clockdrift.TimeSyncData{
					Stratum:    &zero,
					LeapStatus: clockdrift.LeapNotSynchronised,
				},
			}
			sender.Send(input)

			gauge := registry.findByNameAndLabel("clock_drift_reference_info", "reference_id", "unknown")
			Expect(gauge).NotTo(BeNil())
			Expect(gauge.tags["reference_host"]).To(Equal("unknown"))
		})

		DescribeTable("leap_status one-hot encoding", func(active clockdrift.LeapStatus) {
			input := fullInput
			driftCopy := *fullInput.ClockDrift
			driftCopy.LeapStatus = active
			input.ClockDrift = &driftCopy
			sender.Send(input)

			activeLabel := leapLabel(active)
			for _, status := range clockdrift.LeapStatusValues {
				label := leapLabel(status)
				gauge := registry.findByNameAndLabel("clock_drift_leap_status", "status", label)
				Expect(gauge).NotTo(BeNil(), "leap_status gauge with status=%s missing", label)

				expectedVal := 0.0
				if label == activeLabel {
					expectedVal = 1.0
				}
				Expect(gauge.value).To(BeNumerically("==", expectedVal),
					"status=%s expected value %v (active=%s)", label, expectedVal, activeLabel)
			}
		},
			Entry("normal", clockdrift.LeapNormal),
			Entry("insert", clockdrift.LeapInsertSecond),
			Entry("delete", clockdrift.LeapDeleteSecond),
			Entry("unsync", clockdrift.LeapNotSynchronised),
			Entry("unknown", clockdrift.LeapUnknown),
		)

		It("suppresses NaN numeric gauges (parse failures must not become fake-zero)", func() {
			input := collector.SystemStat{
				ClockDriftEnabled: true,
				ClockDrift: &clockdrift.TimeSyncData{
					ReferenceID:         "REF1",
					ReferenceHost:       "1.2.3.4",
					SystemTimeOffsetSec: math.NaN(),
					LastOffsetSec:       math.NaN(),
					RMSOffsetSec:        math.NaN(),
					FrequencyPPM:        math.NaN(),
					ResidualFreqPPM:     math.NaN(),
					SkewPPM:             math.NaN(),
					RootDelaySec:        math.NaN(),
					RootDispersionSec:   math.NaN(),
					UpdateIntervalSec:   math.NaN(),
					LeapStatus:          clockdrift.LeapNormal,
				},
			}
			sender.Send(input)

			suppressed := []string{
				"clock_drift_system_time_offset_seconds",
				"clock_drift_last_offset_seconds",
				"clock_drift_frequency_ppm",
				"clock_drift_root_delay_seconds",
			}
			for _, name := range suppressed {
				Expect(registry.findByName(name)).To(BeNil(), "gauge %s should be suppressed when value is NaN", name)
			}
		})

		It("suppresses the stratum gauge when the source pointer is nil", func() {
			input := collector.SystemStat{
				ClockDriftEnabled: true,
				ClockDrift: &clockdrift.TimeSyncData{
					ReferenceID: "REF1",
					LeapStatus:  clockdrift.LeapNormal,
				},
			}
			sender.Send(input)

			Expect(registry.findByName("clock_drift_stratum")).To(BeNil())
		})

		It("emits clock_drift_collection_errors whenever clock drift collection is enabled", func() {
			input := collector.SystemStat{
				ClockDriftEnabled:     true,
				ClockDriftErrorsTotal: 7,
				ClockDrift:            nil,
			}
			sender.Send(input)

			gauge := registry.findByName("clock_drift_collection_errors")
			Expect(gauge).NotTo(BeNil(), "errors counter must surface even when latest collection failed")
			Expect(gauge.value).To(BeNumerically("==", 7.0))

			Expect(registry.findByName("clock_drift_collection_errors_total")).To(BeNil(),
				"the gauge must NOT carry a _total suffix (Prometheus reserves that for Counter)")
			Expect(registry.findByName("clock_drift_reference_info")).To(BeNil(),
				"reference_info must NOT be emitted when ClockDrift is nil")
			Expect(registry.findByName("clock_drift_stratum")).To(BeNil())
		})

		It("emits no clock drift gauges when ClockDriftEnabled is false (chronyc absent)", func() {
			sender.Send(collector.SystemStat{ClockDriftEnabled: false})

			for key := range registry.gauges {
				Expect(key).NotTo(HavePrefix("clock_drift_"))
			}
		})

		It("emits clock drift gauges even when limitedMetrics is true (PCI-DSS evidence is unconditional)", func() {
			limitedSender := stats.NewPromSender(registry, "test-origin", true, map[string]string{
				"source_id":  "test-origin",
				"deployment": "test-deployment",
				"job":        "test-job",
				"index":      "test-index",
				"ip":         "test-ip",
			})
			limitedSender.Send(fullInput)

			// Spot-check a representative gauge from each clock_drift family.
			Expect(registry.findByName("clock_drift_last_offset_seconds")).NotTo(BeNil(),
				"PCI-DSS audit metric must emit regardless of limitedMetrics")
			Expect(registry.findByName("clock_drift_stratum")).NotTo(BeNil())
			Expect(registry.findByNameAndLabel("clock_drift_leap_status", "status", "normal")).NotTo(BeNil())
			Expect(registry.findByNameAndLabel("clock_drift_reference_info", "reference_id", "REF1")).NotTo(BeNil())
			Expect(registry.findByName("clock_drift_collection_errors")).NotTo(BeNil())
		})
	})

	DescribeTable("does not have disk metrics if disk is not present", func(name, origin, unit string) {
		sender.Send(collector.SystemStat{})

		_, exists := registry.gauges[name+origin+unit]
		Expect(exists).To(BeFalse())
	},
		Entry("system_disk_system_percent", "system_disk_system_percent", "test-origin", "Percent"),
		Entry("system_disk_system_inode_percent", "system_disk_system_inode_percent", "test-origin", "Percent"),
		Entry("system_disk_system_read_bytes", "system_disk_system_read_bytes", "test-origin", "Bytes"),
		Entry("system_disk_system_write_bytes", "system_disk_system_write_bytes", "test-origin", "Bytes"),
		Entry("system_disk_system_read_time", "system_disk_system_read_time", "test-origin", "ms"),
		Entry("system_disk_system_write_time", "system_disk_system_write_time", "test-origin", "ms"),
		Entry("system_disk_system_io_time", "system_disk_system_io_time", "test-origin", "ms"),
		Entry("system_disk_ephemeral_percent", "system_disk_ephemeral_percent", "test-origin", "Percent"),
		Entry("system_disk_ephemeral_inode_percent", "system_disk_ephemeral_inode_percent", "test-origin", "Percent"),
		Entry("system_disk_ephemeral_read_bytes", "system_disk_ephemeral_read_bytes", "test-origin", "Bytes"),
		Entry("system_disk_ephemeral_write_bytes", "system_disk_ephemeral_write_bytes", "test-origin", "Bytes"),
		Entry("system_disk_ephemeral_read_time", "system_disk_ephemeral_read_time", "test-origin", "ms"),
		Entry("system_disk_ephemeral_write_time", "system_disk_ephemeral_write_time", "test-origin", "ms"),
		Entry("system_disk_ephemeral_io_time", "system_disk_ephemeral_io_time", "test-origin", "ms"),
		Entry("system_disk_persistent_percent", "system_disk_persistent_percent", "test-origin", "Percent"),
		Entry("system_disk_persistent_inode_percent", "system_disk_persistent_inode_percent", "test-origin", "Percent"),
		Entry("system_disk_persistent_read_bytes", "system_disk_persistent_read_bytes", "test-origin", "Bytes"),
		Entry("system_disk_persistent_write_bytes", "system_disk_persistent_write_bytes", "test-origin", "Bytes"),
		Entry("system_disk_persistent_read_time", "system_disk_persistent_read_time", "test-origin", "ms"),
		Entry("system_disk_persistent_write_time", "system_disk_persistent_write_time", "test-origin", "ms"),
		Entry("system_disk_persistent_io_time", "system_disk_persistent_io_time", "test-origin", "ms"),
	)

	It("returns 0 for an unhealthy instance", func() {
		sender.Send(unhealthyInstanceInput)

		gauge, exists := registry.gauges["system_healthy"+"test-origin"]
		Expect(exists).To(BeTrue())
		Expect(gauge.value).To(Equal(0.0))
	})

	It("excludes system_healthy if health precence is false", func() {
		sender.Send(collector.SystemStat{})

		_, exists := registry.gauges["system_healthy"+"test-origin"]
		Expect(exists).To(BeFalse())
	})
})

type spyGauge struct {
	name  string
	value float64
	tags  map[string]string
}

func (g *spyGauge) Set(value float64) {
	g.value = value
}

type stubRegistry struct {
	gaugeCount int
	// gauges keys on the simplified label set used by the bulk of the test
	// suite (network_interface and cpu_name only). Clock-drift assertions
	// use findByName / findByNameAndLabel against allGauges to handle the
	// richer label dimensions (status, reference_id, ...).
	gauges    map[string]*spyGauge
	allGauges []*spyGauge
}

func newStubRegistry() *stubRegistry {
	return &stubRegistry{
		gauges: make(map[string]*spyGauge),
	}
}

func (r *stubRegistry) Get(gaugeName, origin, unit string, tags map[string]string) stats.Gauge {
	r.gaugeCount++

	networkName := ""
	if tags != nil {
		networkName = tags["network_interface"]
	}

	cpuName := ""
	if cpuTag, ok := tags["cpu_name"]; ok {
		cpuName = cpuTag
	}

	// Include status / reference_id in the key so multiple clock_drift
	// gauges with the same name but different labels do not overwrite each
	// other in the simplified-key map.
	statusName := ""
	if tags != nil {
		statusName = tags["status"]
	}
	refID := ""
	if tags != nil {
		refID = tags["reference_id"]
	}

	key := gaugeName + origin + unit + networkName + cpuName + statusName + refID

	gauge := &spyGauge{
		name: gaugeName,
		tags: tags,
	}
	r.gauges[key] = gauge
	r.allGauges = append(r.allGauges, gauge)

	return gauge
}

// findByName returns the most recent gauge with the given name (or nil).
// Multiple gauges with the same name but different label sets are possible
// (e.g., clock_drift_leap_status with five status values); in that case the
// caller should use findByNameAndLabel to disambiguate.
func (r *stubRegistry) findByName(name string) *spyGauge {
	for _, g := range r.allGauges {
		if g.name == name {
			return g
		}
	}
	return nil
}

// findByNameAndLabel returns the gauge with the given name whose tags
// include labelKey=labelValue, or nil if none matches.
func (r *stubRegistry) findByNameAndLabel(name, labelKey, labelValue string) *spyGauge {
	for _, g := range r.allGauges {
		if g.name == name && g.tags[labelKey] == labelValue {
			return g
		}
	}
	return nil
}
