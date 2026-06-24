package clockdrift

import (
	"context"
	"errors"
	"io"
	"log"
	"math"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseTimeDirection(t *testing.T) {
	tests := []struct {
		input string
		want  TimeDirection
	}{
		{"slow", DirectionSlow},
		{"fast", DirectionFast},
		{"", DirectionUnknown},
		{"behind", DirectionUnknown},
		{"SLOW", DirectionUnknown},
	}
	for _, tc := range tests {
		got := ParseTimeDirection(tc.input)
		if got != tc.want {
			t.Errorf("ParseTimeDirection(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestParseOffsetWithDirection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantVal float64
		wantDir TimeDirection
	}{
		{"slow flips sign", "0.000314477 seconds slow of NTP time", -0.000314477, DirectionSlow},
		{"fast keeps sign", "0.000314477 seconds fast of NTP time", 0.000314477, DirectionFast},
		{"missing direction", "0.000314477 seconds", 0.000314477, DirectionUnknown},
		{"non-numeric value", "invalid", math.NaN(), DirectionUnknown},
		{"empty string", "", math.NaN(), DirectionUnknown},
	}
	logger := log.New(io.Discard, "", 0)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotVal, gotDir := parseOffsetWithDirection(tc.input, logger, "test")
			if !floatEquals(gotVal, tc.wantVal) || gotDir != tc.wantDir {
				t.Errorf("parseOffsetWithDirection(%q) = (%v, %q), want (%v, %q)",
					tc.input, gotVal, gotDir, tc.wantVal, tc.wantDir)
			}
		})
	}
}

func TestParsePPMWithDirection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantVal float64
		wantDir TimeDirection
	}{
		{"slow flips sign", "10.246 ppm slow", -10.246, DirectionSlow},
		{"fast keeps sign", "10.246 ppm fast", 10.246, DirectionFast},
		{"missing direction", "10.246 ppm", 10.246, DirectionUnknown},
		{"non-numeric value", "junk", math.NaN(), DirectionUnknown},
	}
	logger := log.New(io.Discard, "", 0)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotVal, gotDir := parsePPMWithDirection(tc.input, logger, "test")
			if !floatEquals(gotVal, tc.wantVal) || gotDir != tc.wantDir {
				t.Errorf("parsePPMWithDirection(%q) = (%v, %q), want (%v, %q)",
					tc.input, gotVal, gotDir, tc.wantVal, tc.wantDir)
			}
		})
	}
}

func TestParseLeapStatus(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  LeapStatus
	}{
		{"normal", "Normal", LeapNormal},
		{"insert", "Insert second", LeapInsertSecond},
		{"delete", "Delete second", LeapDeleteSecond},
		{"british unsync", "Not synchronised", LeapNotSynchronised},
		{"american unsync", "Not synchronized", LeapNotSynchronised},
		{"case insensitive", "NORMAL", LeapNormal},
		{"surrounding whitespace", "  Insert second  ", LeapInsertSecond},
		{"empty", "", LeapUnknown},
		{"unknown literal", "Something else", LeapUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseLeapStatus(tc.input)
			if got != tc.want {
				t.Errorf("ParseLeapStatus(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseReferenceID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantID   string
		wantHost string
	}{
		{"id and host", "49B9B6D1 (73.185.182.209)", "73.185.182.209", "73.185.182.209"},
		{"id only no parens", "7F7F0101", "127.127.1.1", ""},
		{"id with empty parens", "00000000 ()", "0.0.0.0", ""},
		{"hostname inside parens", "ABCDEF01 (time.example.com)", "171.205.239.1", "time.example.com"},
		{"stratum 1 GPS ASCII", "47505300 (GPS)", "GPS", "GPS"},
		{"stratum 1 LOCL ASCII", "4c4f434c (LOCL)", "LOCL", "LOCL"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotHost := parseReferenceID(tc.input)
			if gotID != tc.wantID || gotHost != tc.wantHost {
				t.Errorf("parseReferenceID(%q) = (%q, %q), want (%q, %q)",
					tc.input, gotID, gotHost, tc.wantID, tc.wantHost)
			}
		})
	}
}

func TestParseChronyToTimeSyncData_HappyPath(t *testing.T) {
	sample := `Reference ID    : 49B9B6D1 (73.185.182.209)
Stratum         : 4
Ref time (UTC)  : Tue Dec 02 21:14:33 2025
System time     : 0.000314477 seconds slow of NTP time
Last offset     : -0.000364028 seconds
RMS offset      : 0.000758087 seconds
Frequency       : 10.246 ppm slow
Residual freq   : -0.003 ppm
Skew            : 0.146 ppm
Root delay      : 0.070813648 seconds
Root dispersion : 0.011881540 seconds
Update interval : 2073.3 seconds
Leap status     : Normal`

	data, err := parseChronyToTimeSyncData(sample, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.ReferenceID != "73.185.182.209" {
		t.Errorf("ReferenceID: got %q, want %q", data.ReferenceID, "73.185.182.209")
	}
	if data.ReferenceHost != "73.185.182.209" {
		t.Errorf("ReferenceHost: got %q, want %q", data.ReferenceHost, "73.185.182.209")
	}
	if data.Stratum == nil || *data.Stratum != 4 {
		t.Errorf("Stratum: got %v, want pointer to 4", data.Stratum)
	}
	if data.SystemTimeOffsetSec != -0.000314477 {
		t.Errorf("SystemTimeOffsetSec: got %v, want %v", data.SystemTimeOffsetSec, -0.000314477)
	}
	if data.FrequencyPPM != -10.246 {
		t.Errorf("FrequencyPPM: got %v, want %v", data.FrequencyPPM, -10.246)
	}
	if data.LeapStatus != LeapNormal {
		t.Errorf("LeapStatus: got %q, want %q", data.LeapStatus, LeapNormal)
	}
}

func TestParseChronyToTimeSyncData_FatalErrors(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{"empty input", "", "empty"},
		{"only whitespace", "   \n\t\n", "empty"},
		{"chronyc 506 error", "506 Cannot talk to daemon", "chronyc reported an error"},
		{"chronyc 501 error", "501 Not authorised", "chronyc reported an error"},
		{"missing required keys", "Unrelated  : value\nOther    : value", "missing required keys"},
		{"only one required key", "Reference ID  : DEADBEEF", "missing required keys"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseChronyToTimeSyncData(tc.input, nil)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantError)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantError)
			}
		})
	}
}

func TestParseChronyToTimeSyncData_FieldLevelParseFailures(t *testing.T) {
	// Required keys present but several values are unparseable; we expect a
	// successful parse with NaN/nil sentinels rather than fake zeros.
	input := `Reference ID    : 49B9B6D1 (73.185.182.209)
Stratum         : not-a-number
Ref time (UTC)  : garbage
System time     : also-garbage
Last offset     : 0.5 seconds
Leap status     : Normal`

	data, err := parseChronyToTimeSyncData(input, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	if data.Stratum != nil {
		t.Errorf("Stratum: got %v, want nil for unparseable input", data.Stratum)
	}
	if !math.IsNaN(data.SystemTimeOffsetSec) {
		t.Errorf("SystemTimeOffsetSec: got %v, want NaN", data.SystemTimeOffsetSec)
	}
	// Fields that DO parse should still be populated.
	if data.LastOffsetSec != 0.5 {
		t.Errorf("LastOffsetSec: got %v, want 0.5", data.LastOffsetSec)
	}
}

func TestParseChronyToTimeSyncData_CRLFAndPaddedDay(t *testing.T) {
	input := "Reference ID    : 49B9B6D1 (73.185.182.209)\r\n" +
		"Stratum         : 4\r\n" +
		"Ref time (UTC)  : Tue Dec  2 21:14:33 2025\r\n" +
		"Leap status     : Normal\r\n"

	data, err := parseChronyToTimeSyncData(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.RefTimeUTC != "Tue Dec  2 21:14:33 2025" {
		t.Errorf("RefTimeUTC: got %v, want Tue Dec  2 21:14:33 2025", data.RefTimeUTC)
	}
}

func TestSplitKeyValue(t *testing.T) {
	tests := []struct {
		input     string
		wantKey   string
		wantValue string
		wantOK    bool
	}{
		{"Stratum : 4", "Stratum", "4", true},
		{"Reference ID    : 49B9B6D1 (73.185.182.209)", "Reference ID", "49B9B6D1 (73.185.182.209)", true},
		{"Ref time (UTC)  : Tue Dec 02 21:14:33 2025", "Ref time (UTC)", "Tue Dec 02 21:14:33 2025", true},
		{"no colon here", "", "", false},
		{": value-only", "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			gotKey, gotVal, gotOK := splitKeyValue(tc.input)
			if gotOK != tc.wantOK || gotKey != tc.wantKey || gotVal != tc.wantValue {
				t.Errorf("splitKeyValue(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tc.input, gotKey, gotVal, gotOK, tc.wantKey, tc.wantValue, tc.wantOK)
			}
		})
	}
}

func TestChronyBackend_Collect_Success(t *testing.T) {
	sample := `Reference ID    : 49B9B6D1 (73.185.182.209)
Stratum         : 4
Leap status     : Normal`

	b := NewChronyBackend(WithCmdRunner(func(_ context.Context) (string, error) {
		return sample, nil
	}))

	data, err := b.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.ReferenceID != "73.185.182.209" {
		t.Errorf("ReferenceID: got %q, want %q", data.ReferenceID, "73.185.182.209")
	}
	if b.Name() != BackendChrony {
		t.Errorf("Name() = %q, want %q", b.Name(), BackendChrony)
	}
}

func TestChronyBackend_Collect_RunnerError(t *testing.T) {
	wantErr := errors.New("boom")
	b := NewChronyBackend(WithCmdRunner(func(_ context.Context) (string, error) {
		return "", wantErr
	}))

	_, err := b.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestChronyBackend_Collect_HonorsContextCancellation(t *testing.T) {
	b := NewChronyBackend(WithCmdRunner(func(_ context.Context) (string, error) {
		t.Fatal("runner should not be invoked when context is already cancelled")
		return "", nil
	}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := b.Collect(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestChronyBackend_Collect_PropagatesMidExecutionCancellation(t *testing.T) {
	// Runner blocks on the supplied context until cancellation, then returns
	// ctx.Err() the way exec.CommandContext does after SIGKILL.
	b := NewChronyBackend(WithCmdRunner(func(ctx context.Context) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}))

	ctx, cancel := context.WithCancel(context.Background())

	resultCh := make(chan error, 1)
	go func() {
		_, err := b.Collect(ctx)
		resultCh <- err
	}()

	// Give the runner a moment to park on <-ctx.Done() before cancellation
	// so we exercise the mid-execution path, not the pre-cancellation path.
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-resultCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Collect did not return within 1s after context cancellation")
	}

	// A cancelled collection must NOT leave the cache populated: failed
	// snapshots should never be reused as audit evidence in subsequent ticks.
	if cached := b.cached(); cached != nil {
		t.Errorf("expected empty cache after cancelled collect, got %+v", cached)
	}
}

func TestChronyBackend_Collect_CacheReusesResultWithinTTL(t *testing.T) {
	sample := `Reference ID    : ABCD0001 (host)
Stratum         : 2
Leap status     : Normal`

	var calls int32
	b := NewChronyBackend(
		WithCmdRunner(func(_ context.Context) (string, error) {
			atomic.AddInt32(&calls, 1)
			return sample, nil
		}),
		WithCacheTTL(time.Hour),
	)

	for i := 0; i < 5; i++ {
		if _, err := b.Collect(context.Background()); err != nil {
			t.Fatalf("iter %d: unexpected error: %v", i, err)
		}
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("runner invoked %d times, want 1 (cache should suppress)", got)
	}
}

func TestChronyBackend_Collect_CacheDisabledWithZeroTTL(t *testing.T) {
	sample := `Reference ID    : ABCD0001 (host)
Stratum         : 2
Leap status     : Normal`

	var calls int32
	b := NewChronyBackend(
		WithCmdRunner(func(_ context.Context) (string, error) {
			atomic.AddInt32(&calls, 1)
			return sample, nil
		}),
		WithCacheTTL(0),
	)

	for i := 0; i < 3; i++ {
		if _, err := b.Collect(context.Background()); err != nil {
			t.Fatalf("iter %d: unexpected error: %v", i, err)
		}
	}

	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("runner invoked %d times, want 3 (cache should be disabled)", got)
	}
}

// floatEquals treats two NaNs as equal so table-driven tests can express
// "we expect NaN here" without bespoke per-row checks.
func floatEquals(a, b float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	return a == b
}
