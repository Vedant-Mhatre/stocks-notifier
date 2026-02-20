package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
)

func getDirectoryPath() (string, error) {
	// Check if a directory path was provided
	if len(os.Args) < 2 {
		return "", fmt.Errorf("file path not provided")
	}

	// Get the directory path from the command-line argument
	dir := os.Args[1]

	// Check if the directory path is a valid directory
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("'%s' is not a valid directory", dir)
	}

	// Return the directory path
	return dir, nil
}

func directoryPathHelpMessage() {
	fmt.Println("\nProvide the path where the stocks.json file is stored as an argument.\n\nThis file should contain the stock prices at which an alert is to be generated, as shown in the example below:")
	fmt.Println("\nExample stocks.json file:")
	fmt.Println(strings.TrimSpace(`
{
  "TSLA": 220,
  "AAPL": {
    "threshold": 250,
    "direction": "above"
  }
}`))
	fmt.Println("\nOptional flags:")
	fmt.Println("  --web           Start local configuration UI")
	fmt.Println("  --addr=HOST:PORT  Change UI bind address (default 127.0.0.1:8080)")
	fmt.Println("\nCheckout documentation if you need any help:")
	fmt.Println("- https://blog.vmhatre.com/stocks-notifier/")
	fmt.Println("- https://github.com/Vedant-Mhatre/stocks-notifier")
	os.Exit(1)
}

const (
	directionBelow              = "below"
	directionAbove              = "above"
	alertStateFile              = ".stocks-notifier-state.json"
	settingsFile                = ".stocks-notifier-settings.json"
	defaultPollInterval         = 10 * time.Minute
	defaultNearInterval         = 2 * time.Minute
	defaultNearThresholdPercent = 2.0
)

type symbolAlertState struct {
	InAlert          bool  `json:"in_alert"`
	LastNotifiedUnix int64 `json:"last_notified_unix,omitempty"`
}

type AppSettings struct {
	AllowDelayedFallback bool    `json:"allowDelayedFallback"`
	ReminderInterval     string  `json:"reminderInterval,omitempty"`
	PollInterval         string  `json:"pollInterval,omitempty"`
	PollNearInterval     string  `json:"pollNearInterval,omitempty"`
	NearThresholdPercent float64 `json:"nearThresholdPercent,omitempty"`
}

type cliOptions struct {
	Dir  string
	Web  bool
	Addr string
}

var appSettings AppSettings

type AlertRule struct {
	Threshold float64 `json:"threshold"`
	Direction string  `json:"direction,omitempty"`
}

func (rule *AlertRule) normalize() error {
	rule.Direction = strings.ToLower(strings.TrimSpace(rule.Direction))
	if rule.Direction == "" {
		rule.Direction = directionBelow
	}
	if rule.Direction != directionBelow && rule.Direction != directionAbove {
		return fmt.Errorf("unsupported direction %q (supported: %q, %q)", rule.Direction, directionBelow, directionAbove)
	}
	return nil
}

func parseStockRules(rawRules map[string]json.RawMessage) (map[string]AlertRule, error) {
	rules := make(map[string]AlertRule, len(rawRules))

	for symbol, rawRule := range rawRules {
		var legacyThreshold float64
		if err := json.Unmarshal(rawRule, &legacyThreshold); err == nil {
			rules[symbol] = AlertRule{
				Threshold: legacyThreshold,
				Direction: directionBelow,
			}
			continue
		}

		var rule AlertRule
		if err := json.Unmarshal(rawRule, &rule); err != nil {
			return nil, fmt.Errorf("invalid rule for %q: expected a number or object: %v", symbol, err)
		}

		if err := rule.normalize(); err != nil {
			return nil, fmt.Errorf("invalid rule for %q: %v", symbol, err)
		}

		rules[symbol] = rule
	}

	return rules, nil
}

func parseCLIOptions() (cliOptions, error) {
	if len(os.Args) < 2 {
		return cliOptions{}, fmt.Errorf("file path not provided")
	}

	dir := os.Args[1]
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return cliOptions{}, fmt.Errorf("'%s' is not a valid directory", dir)
	}

	opts := cliOptions{
		Dir:  dir,
		Addr: "127.0.0.1:8080",
	}

	for _, arg := range os.Args[2:] {
		switch {
		case arg == "--web":
			opts.Web = true
		case strings.HasPrefix(arg, "--addr="):
			opts.Addr = strings.TrimSpace(strings.TrimPrefix(arg, "--addr="))
			if opts.Addr == "" {
				return cliOptions{}, fmt.Errorf("--addr cannot be empty")
			}
		default:
			return cliOptions{}, fmt.Errorf("unsupported argument %q", arg)
		}
	}

	return opts, nil
}

func readJSONData(dir string) (map[string]AlertRule, error) {

	fullPath := filepath.Join(dir, "stocks.json") //This is required to get platform specific path

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("stocks.json does not exist at given path")
		}
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	var rawRules map[string]json.RawMessage
	if err := decoder.Decode(&rawRules); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("empty file")
		}
		return nil, fmt.Errorf("invalid JSON file: %v", err)
	}

	return parseStockRules(rawRules)
}

func writeJSONData(dir string, rules map[string]AlertRule) error {
	fullPath := filepath.Join(dir, "stocks.json")
	tmpPath := fullPath + ".tmp"

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rules); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, fullPath)
}

func readAppSettings(dir string) (AppSettings, error) {
	fullPath := filepath.Join(dir, settingsFile)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return AppSettings{}, nil
		}
		return AppSettings{}, err
	}
	defer file.Close()

	var settings AppSettings
	if err := json.NewDecoder(file).Decode(&settings); err != nil {
		if err == io.EOF {
			return AppSettings{}, nil
		}
		return AppSettings{}, fmt.Errorf("invalid settings file: %v", err)
	}

	return settings, nil
}

func writeAppSettings(dir string, settings AppSettings) error {
	fullPath := filepath.Join(dir, settingsFile)
	tmpPath := fullPath + ".tmp"

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(settings); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, fullPath)
}

func notify(text string) error {
	iconPath := "assets/warning.png"
	if _, err := os.Stat(iconPath); err != nil {
		iconPath = ""
	}
	if err := beeep.Alert("Stock notifier", text, iconPath); err != nil {
		return fmt.Errorf("notification failed: %w", err)
	}
	return nil
}

func GetStockPrice(symbol string) (float64, error) {
	if symbol == "" {
		return 0, fmt.Errorf("symbol cannot be empty")
	}

	allowDelayed := allowDelayedFallbackEnabled()
	if strings.Contains(symbol, ".") && !allowDelayed {
		log.Printf("Warning: %q looks like a non-US ticker. Real-time quotes only support plain US tickers; set STOCKS_NOTIFIER_ALLOW_DELAYED=1 to use delayed quotes.", symbol)
	}

	if !strings.Contains(symbol, ".") {
		if !allowRealtimeRequest() {
			if allowDelayed {
				delayedPrice, delayedErr := getStooqQuote(symbol)
				if delayedErr == nil {
					return delayedPrice, nil
				}
				return 0, fmt.Errorf("real-time provider temporarily disabled; delayed provider failed: %v", delayedErr)
			}
			return 0, fmt.Errorf("real-time provider temporarily disabled due to recent failures")
		}

		price, err := getStockpricesDevQuote(symbol)
		if err == nil {
			markRealtimeSuccess()
			return price, nil
		}
		markRealtimeFailure(err)

		if allowDelayed {
			delayedPrice, delayedErr := getStooqQuote(symbol)
			if delayedErr == nil {
				return delayedPrice, nil
			}
			return 0, fmt.Errorf("real-time provider failed: %v; delayed provider failed: %v", err, delayedErr)
		}

		return 0, err
	}

	if allowDelayed {
		delayedPrice, delayedErr := getStooqQuote(symbol)
		if delayedErr == nil {
			return delayedPrice, nil
		}
		return 0, fmt.Errorf("delayed provider failed: %v", delayedErr)
	}

	return 0, fmt.Errorf("real-time quotes only support plain US tickers (no suffix). For symbols like %q, set STOCKS_NOTIFIER_ALLOW_DELAYED=1 to use delayed quotes", symbol)
}

const (
	realtimeFailureThreshold = 3
	realtimeCooldown         = 5 * time.Minute
)

var (
	realtimeFailureCount  int
	realtimeDisabledUntil time.Time
)

func allowRealtimeRequest() bool {
	if realtimeDisabledUntil.IsZero() {
		return true
	}
	return time.Now().After(realtimeDisabledUntil)
}

func markRealtimeSuccess() {
	realtimeFailureCount = 0
	realtimeDisabledUntil = time.Time{}
}

func markRealtimeFailure(err error) {
	realtimeFailureCount++
	if realtimeFailureCount >= realtimeFailureThreshold {
		realtimeDisabledUntil = time.Now().Add(realtimeCooldown)
		log.Printf("Real-time provider disabled for %s after %d failures: last error: %v", realtimeCooldown, realtimeFailureCount, err)
	}
}

type stockpricesDevResponse struct {
	Ticker           string   `json:"Ticker"`
	Name             string   `json:"Name"`
	Price            *float64 `json:"Price"`
	ChangeAmount     *float64 `json:"ChangeAmount"`
	ChangePercentage *float64 `json:"ChangePercentage"`
}

func getStockpricesDevQuote(symbol string) (float64, error) {
	cleanSymbol := normalizeStockpricesSymbol(symbol)
	if cleanSymbol == "" {
		return 0, fmt.Errorf("symbol cannot be empty")
	}

	price, err := fetchStockpricesDev(cleanSymbol, "stocks")
	if err == nil {
		return price, nil
	}

	// If it's not a stock symbol, try the ETF endpoint.
	etfPrice, etfErr := fetchStockpricesDev(cleanSymbol, "etfs")
	if etfErr == nil {
		return etfPrice, nil
	}

	return 0, fmt.Errorf("stockprices.dev lookup failed for %q: stocks error: %v; etfs error: %v", cleanSymbol, err, etfErr)
}

func fetchStockpricesDev(symbol, instrument string) (float64, error) {
	url := fmt.Sprintf("https://stockprices.dev/api/%s/%s", instrument, symbol)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build request: %v", err)
	}
	req.Header.Set("User-Agent", "stocks-notifier/1.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch quote for symbol %q: %v", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return 0, fmt.Errorf("unexpected status %d for %q: %s", resp.StatusCode, symbol, msg)
	}

	var payload stockpricesDevResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, fmt.Errorf("failed to decode quote response for %q: %v", symbol, err)
	}

	if payload.Price == nil {
		return 0, fmt.Errorf("missing price for symbol %q", symbol)
	}

	return *payload.Price, nil
}

func normalizeStockpricesSymbol(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return ""
	}
	if dot := strings.Index(symbol, "."); dot != -1 {
		symbol = symbol[:dot]
	}
	return strings.ToUpper(symbol)
}

func getStooqQuote(symbol string) (float64, error) {
	stooqSymbol := normalizeStooqSymbol(symbol)
	if stooqSymbol == "" {
		return 0, fmt.Errorf("symbol cannot be empty")
	}
	url := fmt.Sprintf("https://stooq.com/q/l/?s=%s&i=d", stooqSymbol)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build request: %v", err)
	}
	req.Header.Set("User-Agent", "stocks-notifier/1.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch quote for symbol %q: %v", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status %d fetching quote for %q", resp.StatusCode, symbol)
	}

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("failed to read CSV for %q: %v", symbol, err)
	}
	if len(records) == 0 {
		return 0, fmt.Errorf("empty quote response for %q", symbol)
	}

	header := records[0]
	row := header
	closeIdx := -1

	if len(records) > 1 && len(header) > 0 && strings.EqualFold(strings.TrimSpace(header[0]), "Symbol") {
		row = records[1]
		for i, name := range header {
			if strings.EqualFold(strings.TrimSpace(name), "Close") {
				closeIdx = i
				break
			}
		}
	} else if len(row) >= 7 {
		// Stooq sometimes returns data without a header.
		closeIdx = 6
	}

	if closeIdx == -1 || closeIdx >= len(row) {
		return 0, fmt.Errorf("close price not found for symbol %q", symbol)
	}

	closeVal := strings.TrimSpace(row[closeIdx])
	if closeVal == "" || strings.EqualFold(closeVal, "N/D") {
		return 0, fmt.Errorf("close price unavailable for symbol %q", symbol)
	}

	price, err := strconv.ParseFloat(closeVal, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid close price %q for symbol %q", closeVal, symbol)
	}

	return price, nil
}

func normalizeStooqSymbol(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return ""
	}
	if strings.Contains(symbol, ".") {
		return strings.ToLower(symbol)
	}
	return strings.ToLower(symbol) + ".us"
}

func shouldSendAlert(price float64, rule AlertRule) bool {
	if rule.Direction == directionAbove {
		return price >= rule.Threshold
	}
	return price <= rule.Threshold
}

func readAlertState(dir string) (map[string]symbolAlertState, error) {
	fullPath := filepath.Join(dir, alertStateFile)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]symbolAlertState{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		if err == io.EOF {
			return map[string]symbolAlertState{}, nil
		}
		return nil, fmt.Errorf("invalid alert state file: %v", err)
	}
	state := make(map[string]symbolAlertState, len(raw))
	for symbol, rawValue := range raw {
		var legacy bool
		if err := json.Unmarshal(rawValue, &legacy); err == nil {
			state[symbol] = symbolAlertState{InAlert: legacy}
			continue
		}

		var current symbolAlertState
		if err := json.Unmarshal(rawValue, &current); err != nil {
			return nil, fmt.Errorf("invalid alert state entry for %q: %v", symbol, err)
		}
		state[symbol] = current
	}

	return state, nil
}

func writeAlertState(dir string, state map[string]symbolAlertState) error {
	fullPath := filepath.Join(dir, alertStateFile)
	tmpPath := fullPath + ".tmp"

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(state); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, fullPath)
}

func getReminderIntervalFromEnv() time.Duration {
	return getDurationWithSetting("STOCKS_NOTIFIER_REMINDER_INTERVAL", appSettings.ReminderInterval, 0)
}

func getDurationWithSetting(envKey, settingValue string, defaultValue time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		raw = strings.TrimSpace(settingValue)
	}
	if raw == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed < 0 {
		if defaultValue > 0 {
			log.Printf("Invalid %s value %q, using default %s", envKey, raw, defaultValue)
			return defaultValue
		}
		log.Printf("Invalid %s value %q, feature disabled", envKey, raw)
		return 0
	}
	return parsed
}

func allowDelayedFallbackEnabled() bool {
	raw := strings.TrimSpace(os.Getenv("STOCKS_NOTIFIER_ALLOW_DELAYED"))
	if raw == "" {
		return appSettings.AllowDelayedFallback
	}

	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		log.Printf("Invalid STOCKS_NOTIFIER_ALLOW_DELAYED value %q, using settings/default", raw)
		return appSettings.AllowDelayedFallback
	}
}

func getNearThresholdPercentFromEnv() float64 {
	raw := strings.TrimSpace(os.Getenv("STOCKS_NOTIFIER_NEAR_THRESHOLD_PERCENT"))
	if raw == "" && appSettings.NearThresholdPercent > 0 {
		return appSettings.NearThresholdPercent
	}
	if raw == "" {
		return defaultNearThresholdPercent
	}

	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil || parsed <= 0 {
		log.Printf("Invalid STOCKS_NOTIFIER_NEAR_THRESHOLD_PERCENT value %q, using default %.2f", raw, defaultNearThresholdPercent)
		return defaultNearThresholdPercent
	}

	return parsed
}

func percentDistanceToTrigger(price float64, rule AlertRule) float64 {
	if rule.Threshold <= 0 {
		return 100
	}

	if rule.Direction == directionAbove {
		if price >= rule.Threshold {
			return 0
		}
		return ((rule.Threshold - price) / rule.Threshold) * 100
	}

	if price <= rule.Threshold {
		return 0
	}
	return ((price - rule.Threshold) / rule.Threshold) * 100
}

func determineNextPollInterval(prices map[string]float64, rules map[string]AlertRule, baseInterval, nearInterval time.Duration, nearThresholdPercent float64) (time.Duration, string) {
	if nearInterval <= 0 {
		nearInterval = baseInterval
	}
	if baseInterval <= 0 {
		baseInterval = defaultPollInterval
	}
	if nearThresholdPercent <= 0 {
		nearThresholdPercent = defaultNearThresholdPercent
	}

	if len(prices) == 0 {
		return baseInterval, "no successful quotes"
	}

	for symbol, price := range prices {
		rule, ok := rules[symbol]
		if !ok {
			continue
		}

		if shouldSendAlert(price, rule) {
			return nearInterval, fmt.Sprintf("%s is in alert condition", symbol)
		}

		if percentDistanceToTrigger(price, rule) <= nearThresholdPercent {
			return nearInterval, fmt.Sprintf("%s is near threshold (%.2f%%)", symbol, nearThresholdPercent)
		}
	}

	return baseInterval, "all symbols far from threshold"
}

func shouldNotifyAlert(symbol string, inAlert bool, reminderInterval time.Duration, now time.Time, state map[string]symbolAlertState) bool {
	current := state[symbol]

	if !inAlert {
		state[symbol] = symbolAlertState{InAlert: false}
		return false
	}

	if !current.InAlert {
		state[symbol] = symbolAlertState{InAlert: true, LastNotifiedUnix: now.Unix()}
		return true
	}

	if reminderInterval <= 0 {
		return false
	}

	lastNotified := time.Unix(current.LastNotifiedUnix, 0)
	if current.LastNotifiedUnix == 0 || now.Sub(lastNotified) >= reminderInterval {
		current.LastNotifiedUnix = now.Unix()
		state[symbol] = current
		return true
	}

	return false
}

func pruneAlertState(alertState map[string]symbolAlertState, rules map[string]AlertRule) {
	for symbol := range alertState {
		if _, exists := rules[symbol]; !exists {
			delete(alertState, symbol)
		}
	}
}

func main() {
	opts, err := parseCLIOptions()
	if err != nil {
		fmt.Println(err)
		directoryPathHelpMessage()
	}

	if opts.Web {
		if err := runWebUI(opts.Dir, opts.Addr); err != nil {
			log.Fatalf("Failed to start web UI: %v", err)
		}
		return
	}

	dir := opts.Dir

	settings, err := readAppSettings(dir)
	if err != nil {
		log.Printf("Failed to read settings file, using defaults/env: %v", err)
	}
	appSettings = settings

	alertState, err := readAlertState(dir)
	if err != nil {
		log.Printf("Failed to read alert state, starting fresh: %v", err)
		alertState = map[string]symbolAlertState{}
	}
	reminderInterval := getReminderIntervalFromEnv()
	basePollInterval := getDurationWithSetting("STOCKS_NOTIFIER_POLL_INTERVAL", appSettings.PollInterval, defaultPollInterval)
	nearPollInterval := getDurationWithSetting("STOCKS_NOTIFIER_POLL_NEAR_INTERVAL", appSettings.PollNearInterval, defaultNearInterval)
	nearThresholdPercent := getNearThresholdPercentFromEnv()

	for {
		var stocks map[string]AlertRule
		stocks, err := readJSONData(dir)
		if err != nil {
			if notifyErr := notify(fmt.Sprintf("Error: %v", err)); notifyErr != nil {
				log.Printf("Notify error: %v", notifyErr)
			}
			log.Printf("Error: %v", err)
		}

		prices := make(map[string]float64, len(stocks))
		for symbol, rule := range stocks {

			price, err := GetStockPrice(symbol)
			if err != nil {
				if notifyErr := notify(fmt.Sprintf("Error: %v", err)); notifyErr != nil {
					log.Printf("Notify error: %v", notifyErr)
				}
				log.Printf("Error: %v", err)
				continue
			}

			log.Printf("Price of stock %q: %.2f, Alert is set for price %s %.2f\n", symbol, price, rule.Direction, rule.Threshold)
			prices[symbol] = price

			inAlert := shouldSendAlert(price, rule)
			if shouldNotifyAlert(symbol, inAlert, reminderInterval, time.Now(), alertState) {
				alertMessage := fmt.Sprintf("Price of stock %v: %.2f (target %s %.2f)", symbol, price, rule.Direction, rule.Threshold)
				if notifyErr := notify(alertMessage); notifyErr != nil {
					log.Printf("Notify error: %v", notifyErr)
				}
				// 2 second timeout is needed in MacOS for previous stock notification to get cleared.
				time.Sleep(2 * time.Second)
			}

		}

		pruneAlertState(alertState, stocks)
		if err := writeAlertState(dir, alertState); err != nil {
			log.Printf("Failed to persist alert state: %v", err)
		}

		sleepFor, reason := determineNextPollInterval(prices, stocks, basePollInterval, nearPollInterval, nearThresholdPercent)
		log.Printf("Sleeping for %s (%s)", sleepFor, reason)
		time.Sleep(sleepFor)
	}

}
