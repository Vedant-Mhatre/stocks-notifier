package main

import (
	"encoding/json"
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
