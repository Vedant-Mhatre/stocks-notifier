package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type configPayload struct {
	Rules    map[string]AlertRule `json:"rules"`
	Settings AppSettings          `json:"settings"`
}

type quoteCheckResult struct {
	Symbol string   `json:"symbol"`
	Price  *float64 `json:"price,omitempty"`
	Error  string   `json:"error,omitempty"`
}

func runWebUI(dir, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(webUIHTML))
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetConfig(dir, w)
		case http.MethodPost:
			handleSaveConfig(dir, w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleCheckQuotes(dir, w)
	})

	log.Printf("Stocks Notifier UI available at http://%s", addr)
	return http.ListenAndServe(addr, mux)
}

func handleGetConfig(dir string, w http.ResponseWriter) {
	rules, err := readJSONData(dir)
	if err != nil {
		respondJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	settings, err := readAppSettings(dir)
	if err != nil {
		respondJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, configPayload{
		Rules:    rules,
		Settings: settings,
	})
}

func handleSaveConfig(dir string, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload configPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON payload: %v", err))
		return
	}

	if payload.Rules == nil {
		payload.Rules = map[string]AlertRule{}
	}

	normalizedRules := make(map[string]AlertRule, len(payload.Rules))
	for symbol, rule := range payload.Rules {
		symbol = strings.TrimSpace(strings.ToUpper(symbol))
		if symbol == "" {
			respondJSONError(w, http.StatusBadRequest, "symbol cannot be empty")
			return
		}
		if rule.Threshold <= 0 {
			respondJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid threshold for %s", symbol))
			return
		}
		if err := rule.normalize(); err != nil {
			respondJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid direction for %s: %v", symbol, err))
			return
		}
		normalizedRules[symbol] = rule
	}

	if err := writeJSONData(dir, normalizedRules); err != nil {
		respondJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed writing stocks.json: %v", err))
		return
	}

	if err := writeAppSettings(dir, payload.Settings); err != nil {
		respondJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed writing settings file: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleCheckQuotes(dir string, w http.ResponseWriter) {
	settings, _ := readAppSettings(dir)
	appSettings = settings

	rules, err := readJSONData(dir)
	if err != nil {
		respondJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	results := make([]quoteCheckResult, 0, len(rules))
	for symbol := range rules {
		price, err := GetStockPrice(symbol)
		if err != nil {
			results = append(results, quoteCheckResult{Symbol: symbol, Error: err.Error()})
			continue
		}
		priceCopy := price
		results = append(results, quoteCheckResult{Symbol: symbol, Price: &priceCopy})
	}

	respondJSON(w, http.StatusOK, results)
}

func respondJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondJSONError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, map[string]string{"error": message})
}

const webUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Stocks Notifier Config</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 24px; max-width: 980px; }
    h1, h2 { margin-bottom: 8px; }
    table { width: 100%; border-collapse: collapse; margin: 12px 0; }
    th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
    input, select, button { padding: 8px; font-size: 14px; }
    .row { display: flex; gap: 12px; flex-wrap: wrap; margin: 8px 0; }
    .row label { display: flex; flex-direction: column; gap: 4px; min-width: 220px; }
    .actions { margin-top: 12px; display: flex; gap: 8px; flex-wrap: wrap; }
    .ok { color: #1b7f2a; }
    .err { color: #b42318; }
    .muted { color: #666; font-size: 13px; }
    code { background: #f5f5f5; padding: 2px 4px; }
  </style>
</head>
<body>
  <h1>Stocks Notifier</h1>
  <p class="muted">Local configuration UI. Changes are saved to <code>stocks.json</code> and <code>.stocks-notifier-settings.json</code>.</p>

  <h2>Rules</h2>
  <table id="rulesTable">
    <thead>
      <tr><th>Symbol</th><th>Threshold</th><th>Direction</th><th>Delete</th></tr>
    </thead>
    <tbody></tbody>
  </table>
  <button id="addRuleBtn" type="button">Add Rule</button>

  <h2>Settings</h2>
  <div class="row">
    <label><span>Allow delayed fallback</span><input id="allowDelayedFallback" type="checkbox" /></label>
    <label><span>Reminder interval</span><input id="reminderInterval" placeholder="e.g. 2h" /></label>
    <label><span>Poll interval</span><input id="pollInterval" placeholder="default 10m" /></label>
    <label><span>Near poll interval</span><input id="pollNearInterval" placeholder="default 2m" /></label>
    <label><span>Near threshold percent</span><input id="nearThresholdPercent" type="number" step="0.1" min="0" /></label>
  </div>

  <div class="actions">
    <button id="saveBtn" type="button">Save</button>
    <button id="checkBtn" type="button">Check Quotes Now</button>
  </div>
  <p id="status"></p>
  <pre id="checkOutput"></pre>

  <script>
    const tbody = document.querySelector("#rulesTable tbody");
    const statusEl = document.getElementById("status");
    const checkOutput = document.getElementById("checkOutput");

    function addRuleRow(symbol = "", threshold = "", direction = "below") {
      const tr = document.createElement("tr");
      tr.innerHTML =
        '<td><input data-key="symbol" value="' + symbol + '" /></td>' +
        '<td><input data-key="threshold" type="number" step="0.0001" value="' + threshold + '" /></td>' +
        '<td><select data-key="direction">' +
          '<option value="below"' + (direction === "below" ? " selected" : "") + '>below</option>' +
          '<option value="above"' + (direction === "above" ? " selected" : "") + '>above</option>' +
        '</select></td>' +
        '<td><button type="button" data-action="delete">Delete</button></td>';
      tr.querySelector("[data-action='delete']").addEventListener("click", () => tr.remove());
      tbody.appendChild(tr);
    }

    function setStatus(message, isError = false) {
      statusEl.textContent = message;
      statusEl.className = isError ? "err" : "ok";
    }

    async function loadConfig() {
      const res = await fetch("/api/config");
      const data = await res.json();
      if (!res.ok) {
        setStatus(data.error || "Failed loading config", true);
        return;
      }

      tbody.innerHTML = "";
      Object.entries(data.rules || {}).forEach(([symbol, rule]) => {
        addRuleRow(symbol, rule.threshold, rule.direction || "below");
      });
      if (!Object.keys(data.rules || {}).length) addRuleRow();

      const s = data.settings || {};
      document.getElementById("allowDelayedFallback").checked = !!s.allowDelayedFallback;
      document.getElementById("reminderInterval").value = s.reminderInterval || "";
      document.getElementById("pollInterval").value = s.pollInterval || "";
      document.getElementById("pollNearInterval").value = s.pollNearInterval || "";
      document.getElementById("nearThresholdPercent").value = s.nearThresholdPercent || "";
      setStatus("Configuration loaded");
    }

    function collectPayload() {
      const rules = {};
      [...tbody.querySelectorAll("tr")].forEach((tr) => {
        const symbol = tr.querySelector("[data-key='symbol']").value.trim().toUpperCase();
        const thresholdRaw = tr.querySelector("[data-key='threshold']").value;
        const direction = tr.querySelector("[data-key='direction']").value;
        if (!symbol) return;
        const threshold = parseFloat(thresholdRaw);
        rules[symbol] = { threshold, direction };
      });

      return {
        rules,
        settings: {
          allowDelayedFallback: document.getElementById("allowDelayedFallback").checked,
          reminderInterval: document.getElementById("reminderInterval").value.trim(),
          pollInterval: document.getElementById("pollInterval").value.trim(),
          pollNearInterval: document.getElementById("pollNearInterval").value.trim(),
          nearThresholdPercent: Number(document.getElementById("nearThresholdPercent").value) || 0
        }
      };
    }

    async function saveConfig() {
      const res = await fetch("/api/config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(collectPayload())
      });
      const data = await res.json();
      if (!res.ok) {
        setStatus(data.error || "Save failed", true);
        return;
      }
      setStatus("Configuration saved");
    }

    async function checkQuotes() {
      checkOutput.textContent = "Checking...";
      const res = await fetch("/api/check", { method: "POST" });
      const data = await res.json();
      if (!res.ok) {
        setStatus(data.error || "Quote check failed", true);
        checkOutput.textContent = "";
        return;
      }
      checkOutput.textContent = JSON.stringify(data, null, 2);
      setStatus("Quote check completed");
    }

    document.getElementById("addRuleBtn").addEventListener("click", () => addRuleRow());
    document.getElementById("saveBtn").addEventListener("click", saveConfig);
    document.getElementById("checkBtn").addEventListener("click", checkQuotes);

    loadConfig();
  </script>
</body>
</html>`
