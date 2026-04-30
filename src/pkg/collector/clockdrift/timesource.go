// Package clockdrift collects NTP/chrony time-synchronization data from the
// host for emission as PCI-DSS 10.6 audit evidence.
package clockdrift

import (
	"context"
	"strings"
)

// BackendName identifies a TimeSource implementation. It is exposed as a
// label value on the clock_drift_reference_info gauge so dashboards can
// distinguish backends if more than one ever ships (e.g., w32tm).
const (
	BackendChrony = "chrony"
)

// TimeSource collects time-synchronization data from the system.
//
// Implementations are NOT required to be safe for concurrent use; the
// Collector calls Collect sequentially from a single goroutine. Collect MUST
// honor the supplied context's deadline/cancellation so a wedged time daemon
// cannot stall the broader metrics collection cycle.
type TimeSource interface {
	// Name returns the backend identifier (see BackendChrony).
	Name() string

	// Collect returns the latest time-synchronization snapshot or an error.
	// A nil error and non-nil *TimeSyncData indicates a successful read; the
	// caller may still observe NaN values on individual fields when the
	// underlying tool emitted unparseable output for that field.
	Collect(ctx context.Context) (*TimeSyncData, error)
}

// TimeDirection describes whether the local clock is running slow or fast
// relative to the upstream NTP source.
type TimeDirection string

const (
	DirectionSlow    TimeDirection = "slow"
	DirectionFast    TimeDirection = "fast"
	DirectionUnknown TimeDirection = "unknown"
)

// ParseTimeDirection maps a chronyc direction word to a typed TimeDirection.
// Unknown inputs (including empty string and uppercase variants) return
// DirectionUnknown rather than panicking.
func ParseTimeDirection(s string) TimeDirection {
	switch s {
	case string(DirectionSlow):
		return DirectionSlow
	case string(DirectionFast):
		return DirectionFast
	default:
		return DirectionUnknown
	}
}

// LeapStatus is the chrony-reported leap-second state.
type LeapStatus string

const (
	LeapNormal          LeapStatus = "Normal"
	LeapInsertSecond    LeapStatus = "Insert second"
	LeapDeleteSecond    LeapStatus = "Delete second"
	LeapNotSynchronised LeapStatus = "Not synchronised"
	LeapUnknown         LeapStatus = "unknown"
)

// LeapStatusValues lists every well-known LeapStatus, in dashboard display
// order. It is exported so callers (e.g., the Prometheus sender) can iterate
// without hardcoding the set in two places.
var LeapStatusValues = []LeapStatus{
	LeapNormal,
	LeapInsertSecond,
	LeapDeleteSecond,
	LeapNotSynchronised,
	LeapUnknown,
}

// ParseLeapStatus maps a chronyc leap-status string to a typed LeapStatus.
// Matching is case-insensitive and accepts both British ("Not synchronised")
// and American ("Not synchronized") spellings so a future chrony release that
// normalizes the spelling cannot silently drop the unsync signal.
func ParseLeapStatus(s string) LeapStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case strings.ToLower(string(LeapNormal)):
		return LeapNormal
	case strings.ToLower(string(LeapInsertSecond)):
		return LeapInsertSecond
	case strings.ToLower(string(LeapDeleteSecond)):
		return LeapDeleteSecond
	case strings.ToLower(string(LeapNotSynchronised)), "not synchronized":
		return LeapNotSynchronised
	default:
		return LeapUnknown
	}
}

// TimeSyncData is one snapshot of the local clock's NTP-sync state.
//
// Float fields use math.NaN() as a sentinel for "the underlying tool emitted
// a value we could not parse", which the sender translates to a missing
// gauge data point rather than a misleading zero. Stratum uses *int with nil
// signaling missing/invalid (NTP stratum 0 is a valid sentinel meaning
// "unspecified" and we must not collide with it). RefTimeUnixSec uses
// *float64 because float zero would render as Jan 1 1970 UTC on dashboards.
//
// Sign convention for offset and PPM fields: when the chrony direction word
// is "slow" the parser stores the value as negative; "fast" stays positive.
// The companion Direction enum preserves the original word so consumers can
// disambiguate without relying on the sign.
type TimeSyncData struct {
	ReferenceID         string        `json:"reference_id"`
	ReferenceHost       string        `json:"reference_host"`
	Stratum             *int          `json:"stratum,omitempty"`
	RefTimeUTC          string        `json:"ref_time_utc"`
	RefTimeUnixSec      *float64      `json:"ref_time_unix_sec,omitempty"`
	SystemTimeOffsetSec float64       `json:"system_time_offset_sec"`
	SystemTimeDirection TimeDirection `json:"system_time_direction"`
	LastOffsetSec       float64       `json:"last_offset_sec"`
	RMSOffsetSec        float64       `json:"rms_offset_sec"`
	FrequencyPPM        float64       `json:"frequency_ppm"`
	FrequencyDirection  TimeDirection `json:"frequency_direction"`
	ResidualFreqPPM     float64       `json:"residual_freq_ppm"`
	SkewPPM             float64       `json:"skew_ppm"`
	RootDelaySec        float64       `json:"root_delay_sec"`
	RootDispersionSec   float64       `json:"root_dispersion_sec"`
	UpdateIntervalSec   float64       `json:"update_interval_sec"`
	LeapStatus          LeapStatus    `json:"leap_status"`
}
