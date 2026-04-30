package clockdrift

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// chronyc field-name keys exactly as printed by `chronyc tracking`. Promoted
// to constants so a typo is a compile error rather than a silently-zeroed
// metric.
const (
	keyReferenceID    = "Reference ID"
	keyStratum        = "Stratum"
	keyRefTimeUTC     = "Ref time (UTC)"
	keySystemTime     = "System time"
	keyLastOffset     = "Last offset"
	keyRMSOffset      = "RMS offset"
	keyFrequency      = "Frequency"
	keyResidualFreq   = "Residual freq"
	keySkew           = "Skew"
	keyRootDelay      = "Root delay"
	keyRootDispersion = "Root dispersion"
	keyUpdateInterval = "Update interval"
	keyLeapStatus     = "Leap status"
)

// requiredKeys are the minimum chronyc fields that MUST appear in valid
// tracking output. Their absence indicates either a chronyc error response
// (e.g., "506 Cannot talk to daemon") or a parser regression after a chrony
// version bump; either way, returning a fatal error is preferable to
// emitting a fully-zeroed snapshot that an auditor would read as healthy.
var requiredKeys = []string{keyReferenceID, keyStratum, keyLeapStatus}

// chronyErrorPrefixes are line prefixes that chronyc uses to report client
// errors. These never appear in successful `tracking` output so detecting
// any of them lets us fail fast with a useful diagnostic.
var chronyErrorPrefixes = []string{"506 ", "501 ", "Error :", "Could not"}

// DefaultCacheTTL is how long ChronyBackend reuses the most recent successful
// snapshot before invoking chronyc again. Compliance dashboards stay fresh
// while we avoid forking once per system-metrics tick (typically 15s).
const DefaultCacheTTL = 0 //5 * time.Minute

// ChronyBackend implements TimeSource by executing `chronyc tracking`.
//
// Behavior:
//   - exec is wrapped in the caller-supplied context (5s timeout in
//     production) so a wedged chronyd cannot stall the metrics agent.
//   - Locale and timezone are pinned via LANG=C, LC_ALL=C, TZ=UTC so a
//     reconfigured stemcell cannot accidentally rename chrony's field keys
//     out from under the parser.
//   - Successful results are cached for cacheTTL to avoid forking chronyc
//     more often than the audit cadence demands.
//
// Linux is the only currently-supported platform. Azure Linux VMs use chrony
// with a PTP reference clock (/dev/ptp_hyperv) and produce identical
// `chronyc tracking` output, so no Azure-specific code is required.
type ChronyBackend struct {
	runner   func(ctx context.Context) (string, error)
	logger   *log.Logger
	cacheTTL time.Duration

	mu         sync.Mutex
	cachedData *TimeSyncData
	cachedAt   time.Time
}

// ChronyOption configures a ChronyBackend at construction time.
type ChronyOption func(*ChronyBackend)

// WithCmdRunner overrides the function used to execute `chronyc tracking`.
// Tests inject a fake to return canned output without forking a process.
func WithCmdRunner(r func(ctx context.Context) (string, error)) ChronyOption {
	return func(c *ChronyBackend) {
		c.runner = r
	}
}

// WithLogger sets the logger used for non-fatal parse warnings. If unset,
// warnings are silently discarded so the package never spams the agent log.
func WithLogger(l *log.Logger) ChronyOption {
	return func(c *ChronyBackend) {
		c.logger = l
	}
}

// WithCacheTTL overrides the duration for which a successful snapshot is
// reused. A non-positive value disables caching and forces every Collect
// call to invoke chronyc.
func WithCacheTTL(d time.Duration) ChronyOption {
	return func(c *ChronyBackend) {
		c.cacheTTL = d
	}
}

// NewChronyBackend constructs a ChronyBackend with sensible defaults that
// can be selectively overridden via options.
func NewChronyBackend(opts ...ChronyOption) *ChronyBackend {
	b := &ChronyBackend{
		runner:   execChronyc,
		logger:   log.New(io.Discard, "", 0),
		cacheTTL: DefaultCacheTTL,
	}
	for _, o := range opts {
		o(b)
	}
	if b.runner == nil {
		b.runner = execChronyc
	}
	if b.logger == nil {
		b.logger = log.New(io.Discard, "", 0)
	}
	return b
}

// Name returns the backend identifier ("chrony").
func (c *ChronyBackend) Name() string { return BackendChrony }

// Collect returns the latest TimeSyncData, honoring the supplied context's
// deadline. The result may come from an in-memory cache if the previous call
// succeeded within cacheTTL.
func (c *ChronyBackend) Collect(ctx context.Context) (*TimeSyncData, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if cached := c.cached(); cached != nil {
		return cached, nil
	}

	output, err := c.runner(ctx)
	if err != nil {
		return nil, fmt.Errorf("chronyc tracking: %w", err)
	}

	data, err := parseChronyToTimeSyncData(output, c.logger)
	if err != nil {
		return nil, err
	}

	c.store(data)
	return data, nil
}

func (c *ChronyBackend) cached() *TimeSyncData {
	if c.cacheTTL <= 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cachedData == nil || time.Since(c.cachedAt) > c.cacheTTL {
		return nil
	}
	return c.cachedData
}

func (c *ChronyBackend) store(data *TimeSyncData) {
	if c.cacheTTL <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cachedData = data
	c.cachedAt = time.Now()
}

// execChronyc runs `chronyc tracking` under the supplied context, with the
// process environment trimmed to a known-good locale and timezone so chrony
// produces field keys in the format the parser expects.
func execChronyc(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "chronyc", "tracking")
	// LANG=C and LC_ALL=C are load-bearing: chrony's field keys ("Reference
	// ID", "Leap status", ...) are localizable via the LC_TIME locale, and a
	// reconfigured stemcell could otherwise rename them out from under the
	// parser. TZ=UTC is belt-and-suspenders only -- chronyc tracking already
	// formats "Ref time (UTC)" via gmtime_r regardless of TZ.
	cmd.Env = append(os.Environ(), "LANG=C", "LC_ALL=C", "TZ=UTC")

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("chronyc exited %d: %s", exitErr.ExitCode(), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("failed to run chronyc: %w", err)
	}
	return string(out), nil
}

// refIDPattern matches `<hex-id> (<host>)`. The host group is permissive
// because chrony emits IPv4, IPv6, and DNS hostnames.
var refIDPattern = regexp.MustCompile(`^([0-9A-Fa-f]+)\s*\((.*)\)$`)

// parseChronyToTimeSyncData parses raw `chronyc tracking` output into typed
// fields. Field-level parse failures populate math.NaN() (for floats) or nil
// (for *int / *float64) so the sender can suppress the corresponding gauges.
// Whole-output failures (chronyc error response, missing required keys)
// return an error so the caller can surface them via the error counter.
func parseChronyToTimeSyncData(output string, logger *log.Logger) (*TimeSyncData, error) {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil, errors.New("empty chronyc tracking output")
	}

	for _, prefix := range chronyErrorPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return nil, fmt.Errorf("chronyc reported an error: %s", firstLine(trimmed))
		}
	}

	raw := make(map[string]string)
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := splitKeyValue(line)
		if !ok {
			continue
		}
		raw[key] = value
	}

	if len(raw) == 0 {
		return nil, errors.New("no valid key-value pairs found in chronyc output")
	}

	missing := missingRequiredKeys(raw)
	if len(missing) > 0 {
		return nil, fmt.Errorf("chronyc output missing required keys: %s", strings.Join(missing, ", "))
	}

	data := &TimeSyncData{
		SystemTimeOffsetSec: math.NaN(),
		LastOffsetSec:       math.NaN(),
		RMSOffsetSec:        math.NaN(),
		FrequencyPPM:        math.NaN(),
		ResidualFreqPPM:     math.NaN(),
		SkewPPM:             math.NaN(),
		RootDelaySec:        math.NaN(),
		RootDispersionSec:   math.NaN(),
		UpdateIntervalSec:   math.NaN(),
		SystemTimeDirection: DirectionUnknown,
		FrequencyDirection:  DirectionUnknown,
	}

	if v, ok := raw[keyReferenceID]; ok {
		data.ReferenceID, data.ReferenceHost = parseReferenceID(v)
	}

	if v, ok := raw[keyStratum]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			data.Stratum = &n
		} else {
			logger.Printf("clockdrift: parse Stratum %q: %v", v, err)
		}
	}

	data.RefTimeUTC = raw[keyRefTimeUTC]
	if ts, ok := parseRefTime(data.RefTimeUTC); ok {
		data.RefTimeUnixSec = &ts
	} else if data.RefTimeUTC != "" {
		logger.Printf("clockdrift: parse RefTimeUTC %q failed", data.RefTimeUTC)
	}

	data.LeapStatus = ParseLeapStatus(raw[keyLeapStatus])

	data.SystemTimeOffsetSec, data.SystemTimeDirection = parseOffsetWithDirection(raw[keySystemTime], logger, keySystemTime)
	data.LastOffsetSec = parseSeconds(raw[keyLastOffset], logger, keyLastOffset)
	data.RMSOffsetSec = parseSeconds(raw[keyRMSOffset], logger, keyRMSOffset)
	data.FrequencyPPM, data.FrequencyDirection = parsePPMWithDirection(raw[keyFrequency], logger, keyFrequency)
	data.ResidualFreqPPM = parsePPM(raw[keyResidualFreq], logger, keyResidualFreq)
	data.SkewPPM = parsePPM(raw[keySkew], logger, keySkew)
	data.RootDelaySec = parseSeconds(raw[keyRootDelay], logger, keyRootDelay)
	data.RootDispersionSec = parseSeconds(raw[keyRootDispersion], logger, keyRootDispersion)
	data.UpdateIntervalSec = parseSeconds(raw[keyUpdateInterval], logger, keyUpdateInterval)

	return data, nil
}

// splitKeyValue cuts a chronyc tracking line into (key, value) at the first
// colon. Splitting on the first colon (rather than the literal " : "
// separator) tolerates column-alignment changes and any quirks in chrony's
// padding logic.
func splitKeyValue(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

func missingRequiredKeys(raw map[string]string) []string {
	var missing []string
	for _, k := range requiredKeys {
		if _, ok := raw[k]; !ok {
			missing = append(missing, k)
		}
	}
	return missing
}

// parseReferenceID splits a chronyc reference value of the form
// `<hex-id> (<host>)` into (id, host). When the host segment is missing or
// empty the host return is empty and parentheses never leak into label
// values.
func parseReferenceID(v string) (string, string) {
	v = strings.TrimSpace(v)
	if m := refIDPattern.FindStringSubmatch(v); m != nil {
		return m[1], strings.TrimSpace(m[2])
	}
	// Strip any stray parens (e.g., `7F7F0101 ()` when chrony has no peer)
	// rather than letting them propagate into Prometheus label values.
	v = strings.TrimRight(strings.TrimSpace(strings.TrimSuffix(v, "()")), " ")
	return v, ""
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func parseSeconds(s string, logger *log.Logger, fieldName string) float64 {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return math.NaN()
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		logger.Printf("clockdrift: parse %s %q: %v", fieldName, s, err)
		return math.NaN()
	}
	return v
}

func parseOffsetWithDirection(s string, logger *log.Logger, fieldName string) (float64, TimeDirection) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return math.NaN(), DirectionUnknown
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		logger.Printf("clockdrift: parse %s %q: %v", fieldName, s, err)
		return math.NaN(), DirectionUnknown
	}
	if len(fields) < 3 {
		return v, DirectionUnknown
	}
	dir := ParseTimeDirection(fields[2])
	if dir == DirectionSlow {
		v = -v
	}
	return v, dir
}

func parsePPMWithDirection(s string, logger *log.Logger, fieldName string) (float64, TimeDirection) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return math.NaN(), DirectionUnknown
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		logger.Printf("clockdrift: parse %s %q: %v", fieldName, s, err)
		return math.NaN(), DirectionUnknown
	}
	if len(fields) < 3 {
		return v, DirectionUnknown
	}
	dir := ParseTimeDirection(fields[2])
	if dir == DirectionSlow {
		v = -v
	}
	return v, dir
}

func parsePPM(s string, logger *log.Logger, fieldName string) float64 {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return math.NaN()
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		logger.Printf("clockdrift: parse %s %q: %v", fieldName, s, err)
		return math.NaN()
	}
	return v
}

// parseRefTime converts chronyc's "Ref time (UTC)" value to a Unix epoch
// timestamp. The layout uses `_2` (space-padded day) so values like
// "Tue Dec  2 21:14:33 2025" parse correctly. The bool return distinguishes
// "successfully parsed" from "missing or unparseable" so the caller can
// represent the latter as a nil pointer rather than the misleading 1970
// epoch.
func parseRefTime(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	layouts := []string{
		"Mon Jan _2 15:04:05 2006",
		"Mon Jan 02 15:04:05 2006",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return float64(t.Unix()), true
		}
	}
	return 0, false
}
