package clockdrift

import (
	"context"
	"encoding/hex"
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
	keyFrequency      = "Frequency"
	keyRootDelay      = "Root delay"
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

// DefaultCacheTTL controls result reuse between collection ticks. Zero
// disables caching: chronyc is invoked on every tick (typically every 15s),
// keeping clock drift metrics at the same cadence as all other system metrics
// and ensuring a lost-sync event surfaces within one tick rather than up to
// five minutes later.
const DefaultCacheTTL = 0

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
		FrequencyPPM:        math.NaN(),
		RootDelaySec:        math.NaN(),
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

	data.LeapStatus = ParseLeapStatus(raw[keyLeapStatus])

	data.SystemTimeOffsetSec, data.SystemTimeDirection = parseOffsetWithDirection(raw[keySystemTime], logger, keySystemTime)
	data.LastOffsetSec = parseSeconds(raw[keyLastOffset], logger, keyLastOffset)
	data.FrequencyPPM, data.FrequencyDirection = parsePPMWithDirection(raw[keyFrequency], logger, keyFrequency)
	data.RootDelaySec = parseSeconds(raw[keyRootDelay], logger, keyRootDelay)

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
// `<hex-id> (<host>)` into (id, host), decoding the hexadecimal <hex-id> into
// an IP address or an ASCII string if it is a stratum 1 source. When the host
// segment is missing or empty the host return is empty and parentheses never
// leak into label values.
func parseReferenceID(v string) (string, string) {
	v = strings.TrimSpace(v)
	if m := refIDPattern.FindStringSubmatch(v); m != nil {
		return decodeReferenceID(m[1]), strings.TrimSpace(m[2])
	}
	// Strip any stray parens (e.g., `7F7F0101 ()` when chrony has no peer)
	// rather than letting them propagate into Prometheus label values.
	v = strings.TrimRight(strings.TrimSpace(strings.TrimSuffix(v, "()")), " ")
	return decodeReferenceID(v), ""
}

// decodeReferenceID converts a 32-bit hexadecimal reference ID string
// to either a human-readable ASCII string (if it's a stratum 1 reference
// clock like GPS/PPS) or an IPv4 address. If the input is not a valid
// 8-character hex string, it is returned unmodified.
func decodeReferenceID(hexStr string) string {
	hexStr = strings.TrimSpace(hexStr)
	if len(hexStr) != 8 {
		return hexStr
	}
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return hexStr
	}

	// Check if all bytes are printable ASCII or trailing null padding.
	// This helps identify Stratum 1 reference clocks (e.g., GPS, PPS, ACTS, LOCL).
	isASCII := true
	for i, b := range bytes {
		if b == 0 {
			// Zero padding must extend to the end of the 4-byte block
			for j := i; j < len(bytes); j++ {
				if bytes[j] != 0 {
					isASCII = false
					break
				}
			}
			break
		}
		// Printable ASCII characters are 32 (' ') through 126 ('~')
		if b < 32 || b > 126 {
			isASCII = false
			break
		}
	}

	if isASCII {
		s := string(bytes)
		s = strings.TrimRight(s, "\x00")
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			return s
		}
	}

	// Otherwise, format as an IPv4 address
	return fmt.Sprintf("%d.%d.%d.%d", bytes[0], bytes[1], bytes[2], bytes[3])
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
