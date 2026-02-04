package main

import "testing"

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
