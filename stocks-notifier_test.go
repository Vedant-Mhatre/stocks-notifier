package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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

func TestShouldNotifyOnStateChange(t *testing.T) {
	state := map[string]bool{}

	if !shouldNotifyOnStateChange("AAPL", true, state) {
		t.Fatalf("first entry into alert should notify")
	}
	if shouldNotifyOnStateChange("AAPL", true, state) {
		t.Fatalf("same alert state should not notify repeatedly")
	}
	if shouldNotifyOnStateChange("AAPL", false, state) {
		t.Fatalf("exiting alert should not notify")
	}
	if !shouldNotifyOnStateChange("AAPL", true, state) {
		t.Fatalf("re-entering alert should notify again")
	}
}

func TestReadWriteAlertState(t *testing.T) {
	dir := t.TempDir()
	input := map[string]bool{
		"AAPL": true,
		"TSLA": false,
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
