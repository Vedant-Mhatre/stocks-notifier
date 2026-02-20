package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizeStockpricesSymbol(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "empty", input: "", expect: ""},
		{name: "plain", input: "tsla", expect: "TSLA"},
		{name: "trim", input: "  aapl ", expect: "AAPL"},
		{name: "suffix", input: "INFY.NS", expect: "INFY"},
	}

	for _, tt := range tests {
		if got := normalizeStockpricesSymbol(tt.input); got != tt.expect {
			t.Fatalf("%s: expected %q, got %q", tt.name, tt.expect, got)
		}
	}
}

func TestNormalizeStooqSymbol(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "empty", input: "", expect: ""},
		{name: "plain", input: "TSLA", expect: "tsla.us"},
		{name: "suffix", input: "INFY.NS", expect: "infy.ns"},
		{name: "trim", input: "  BRK.B ", expect: "brk.b"},
	}

	for _, tt := range tests {
		if got := normalizeStooqSymbol(tt.input); got != tt.expect {
			t.Fatalf("%s: expected %q, got %q", tt.name, tt.expect, got)
		}
	}
}

func TestParseStockRulesSupportsLegacyAndDirectional(t *testing.T) {
	payload := []byte(`{
		"AAPL": 180,
		"TSLA": {"threshold": 250, "direction": "above"},
		"MSFT": {"threshold": 300}
	}`)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("failed to unmarshal test payload: %v", err)
	}

	rules, err := parseStockRules(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if rules["AAPL"].Direction != directionBelow || rules["AAPL"].Threshold != 180 {
		t.Fatalf("legacy format should default to below: %#v", rules["AAPL"])
	}
	if rules["TSLA"].Direction != directionAbove || rules["TSLA"].Threshold != 250 {
		t.Fatalf("directional format not parsed correctly: %#v", rules["TSLA"])
	}
	if rules["MSFT"].Direction != directionBelow || rules["MSFT"].Threshold != 300 {
		t.Fatalf("missing direction should default to below: %#v", rules["MSFT"])
	}
}

func TestParseStockRulesRejectsInvalidDirection(t *testing.T) {
	payload := []byte(`{"TSLA": {"threshold": 250, "direction": "sideways"}}`)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("failed to unmarshal test payload: %v", err)
	}

	_, err := parseStockRules(raw)
	if err == nil {
		t.Fatalf("expected invalid direction error")
	}
}

func TestShouldSendAlert(t *testing.T) {
	if !shouldSendAlert(100, AlertRule{Threshold: 110, Direction: directionBelow}) {
		t.Fatalf("below alert should trigger when price is lower")
	}
	if shouldSendAlert(120, AlertRule{Threshold: 110, Direction: directionBelow}) {
		t.Fatalf("below alert should not trigger when price is higher")
	}
	if !shouldSendAlert(120, AlertRule{Threshold: 110, Direction: directionAbove}) {
		t.Fatalf("above alert should trigger when price is higher")
	}
	if shouldSendAlert(100, AlertRule{Threshold: 110, Direction: directionAbove}) {
		t.Fatalf("above alert should not trigger when price is lower")
	}
}

func TestShouldNotifyAlertStateTransitions(t *testing.T) {
	state := map[string]symbolAlertState{}
	now := time.Unix(1_700_000_000, 0)

	if !shouldNotifyAlert("AAPL", true, 0, now, state) {
		t.Fatalf("first entry into alert should notify")
	}
	if shouldNotifyAlert("AAPL", true, 0, now.Add(time.Minute), state) {
		t.Fatalf("same alert state should not notify repeatedly when reminders are disabled")
	}
	if shouldNotifyAlert("AAPL", false, 0, now.Add(2*time.Minute), state) {
		t.Fatalf("exiting alert should not notify")
	}
	if !shouldNotifyAlert("AAPL", true, 0, now.Add(3*time.Minute), state) {
		t.Fatalf("re-entering alert should notify again")
	}
}

func TestShouldNotifyAlertReminderInterval(t *testing.T) {
	state := map[string]symbolAlertState{}
	now := time.Unix(1_700_000_000, 0)

	if !shouldNotifyAlert("AAPL", true, 2*time.Hour, now, state) {
		t.Fatalf("first entry should notify")
	}
	if shouldNotifyAlert("AAPL", true, 2*time.Hour, now.Add(time.Hour), state) {
		t.Fatalf("should not remind before interval")
	}
	if !shouldNotifyAlert("AAPL", true, 2*time.Hour, now.Add(2*time.Hour), state) {
		t.Fatalf("should remind when interval elapses")
	}
}

func TestReadWriteAlertState(t *testing.T) {
	dir := t.TempDir()
	input := map[string]symbolAlertState{
		"AAPL": {InAlert: true, LastNotifiedUnix: 1_700_000_000},
		"TSLA": {InAlert: false},
	}

	if err := writeAlertState(dir, input); err != nil {
		t.Fatalf("writeAlertState failed: %v", err)
	}

	output, err := readAlertState(dir)
	if err != nil {
		t.Fatalf("readAlertState failed: %v", err)
	}

	if len(output) != len(input) {
		t.Fatalf("state length mismatch: got %d want %d", len(output), len(input))
	}
	for symbol, expected := range input {
		if output[symbol] != expected {
			t.Fatalf("state mismatch for %s: got %v want %v", symbol, output[symbol], expected)
		}
	}
}

func TestReadAlertStateLegacyFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, alertStateFile)
	if err := os.WriteFile(path, []byte(`{"AAPL":true,"TSLA":false}`), 0644); err != nil {
		t.Fatalf("failed to write legacy state file: %v", err)
	}

	state, err := readAlertState(dir)
	if err != nil {
		t.Fatalf("readAlertState failed for legacy format: %v", err)
	}

	if !state["AAPL"].InAlert || state["TSLA"].InAlert {
		t.Fatalf("legacy state not parsed correctly: %#v", state)
	}
}

func TestReadAlertStateMissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	state, err := readAlertState(dir)
	if err != nil {
		t.Fatalf("readAlertState should not fail for missing file: %v", err)
	}
	if len(state) != 0 {
		t.Fatalf("expected empty state for missing file, got: %#v", state)
	}
}

func TestReadAlertStateInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, alertStateFile)
	if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatalf("failed to write invalid state file: %v", err)
	}

	_, err := readAlertState(dir)
	if err == nil {
		t.Fatalf("expected invalid JSON error")
	}
}

func TestGetReminderIntervalFromEnv(t *testing.T) {
	_ = os.Unsetenv("STOCKS_NOTIFIER_REMINDER_INTERVAL")
	if got := getReminderIntervalFromEnv(); got != 0 {
		t.Fatalf("expected zero interval when env missing, got: %v", got)
	}

	if err := os.Setenv("STOCKS_NOTIFIER_REMINDER_INTERVAL", "90m"); err != nil {
		t.Fatalf("failed setting env: %v", err)
	}
	if got := getReminderIntervalFromEnv(); got != 90*time.Minute {
		t.Fatalf("expected 90m, got: %v", got)
	}

	if err := os.Setenv("STOCKS_NOTIFIER_REMINDER_INTERVAL", "invalid"); err != nil {
		t.Fatalf("failed setting env: %v", err)
	}
	if got := getReminderIntervalFromEnv(); got != 0 {
		t.Fatalf("expected zero interval for invalid env, got: %v", got)
	}
}

func TestPercentDistanceToTrigger(t *testing.T) {
	belowRule := AlertRule{Threshold: 100, Direction: directionBelow}
	aboveRule := AlertRule{Threshold: 100, Direction: directionAbove}

	if got := percentDistanceToTrigger(95, belowRule); got != 0 {
		t.Fatalf("expected zero distance when below-rule is already in alert, got: %v", got)
	}
	if got := percentDistanceToTrigger(105, belowRule); got != 5 {
		t.Fatalf("expected 5 percent distance, got: %v", got)
	}
	if got := percentDistanceToTrigger(105, aboveRule); got != 0 {
		t.Fatalf("expected zero distance when above-rule is already in alert, got: %v", got)
	}
	if got := percentDistanceToTrigger(95, aboveRule); got != 5 {
		t.Fatalf("expected 5 percent distance, got: %v", got)
	}
}

func TestDetermineNextPollInterval(t *testing.T) {
	base := 10 * time.Minute
	near := 2 * time.Minute
	nearPct := 2.0
	rules := map[string]AlertRule{
		"AAPL": {Threshold: 100, Direction: directionBelow},
	}

	tests := []struct {
		name         string
		prices       map[string]float64
		expect       time.Duration
		expectReason string
	}{
		{
			name:         "no prices uses base",
			prices:       map[string]float64{},
			expect:       base,
			expectReason: "no successful quotes",
		},
		{
			name:         "in alert uses near",
			prices:       map[string]float64{"AAPL": 99},
			expect:       near,
			expectReason: "AAPL is in alert condition",
		},
		{
			name:         "near threshold uses near",
			prices:       map[string]float64{"AAPL": 101},
			expect:       near,
			expectReason: "AAPL is near threshold (2.00%)",
		},
		{
			name:         "far threshold uses base",
			prices:       map[string]float64{"AAPL": 110},
			expect:       base,
			expectReason: "all symbols far from threshold",
		},
	}

	for _, tt := range tests {
		gotInterval, gotReason := determineNextPollInterval(tt.prices, rules, base, near, nearPct)
		if gotInterval != tt.expect {
			t.Fatalf("%s: expected interval %v, got %v", tt.name, tt.expect, gotInterval)
		}
		if gotReason != tt.expectReason {
			t.Fatalf("%s: expected reason %q, got %q", tt.name, tt.expectReason, gotReason)
		}
	}
}
