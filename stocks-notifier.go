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
  "ICICIBANK": 880,
  "HDFCBANK": 1600,
  "INFY": 1500
}`))
	fmt.Println("\nCheckout documentation if you need any help: https://blog.vmhatre.com/stocks-notifier/")
	os.Exit(1)
}

func readJSONData(dir string) (map[string]float64, error) {

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

	var data map[string]float64
	if err := decoder.Decode(&data); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("empty file")
		}
		return nil, fmt.Errorf("invalid JSON file: %v", err)
	}

	return data, nil
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

	if strings.Contains(symbol, ".") && os.Getenv("STOCKS_NOTIFIER_ALLOW_DELAYED") != "1" {
		log.Printf("Warning: %q looks like a non-US ticker. Real-time quotes only support plain US tickers; set STOCKS_NOTIFIER_ALLOW_DELAYED=1 to use delayed quotes.", symbol)
	}

	if !allowRealtimeRequest() {
		return 0, fmt.Errorf("real-time provider temporarily disabled due to recent failures")
	}

	if !strings.Contains(symbol, ".") {
		price, err := getStockpricesDevQuote(symbol)
		if err == nil {
			markRealtimeSuccess()
			return price, nil
		}
		markRealtimeFailure(err)

		if os.Getenv("STOCKS_NOTIFIER_ALLOW_DELAYED") == "1" {
			delayedPrice, delayedErr := getStooqQuote(symbol)
			if delayedErr == nil {
				return delayedPrice, nil
			}
			return 0, fmt.Errorf("real-time provider failed: %v; delayed provider failed: %v", err, delayedErr)
		}

		return 0, err
	}

	if os.Getenv("STOCKS_NOTIFIER_ALLOW_DELAYED") == "1" {
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
	realtimeFailureCount int
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

func main() {

	dir, error := getDirectoryPath()
	if error != nil {
		fmt.Println(error)
		directoryPathHelpMessage()
	}

	for {
		var stocks map[string]float64
		stocks, err := readJSONData(dir)
		if err != nil {
			if notifyErr := notify(fmt.Sprintf("Error: %v", err)); notifyErr != nil {
				log.Printf("Notify error: %v", notifyErr)
			}
			log.Printf("Error: %v", err)
		}

		for symbol, alertPrice := range stocks {

			price, err := GetStockPrice(symbol)
			if err != nil {
				if notifyErr := notify(fmt.Sprintf("Error: %v", err)); notifyErr != nil {
					log.Printf("Notify error: %v", notifyErr)
				}
				log.Printf("Error: %v", err)
				continue
			}

			log.Printf("Price of stock %q: %.2f, Alert is set at %.2f\n", symbol, price, alertPrice)

			if price <= alertPrice {
				alertMessage := fmt.Sprintf("Price of stock %v: %.2f", symbol, price)
				if notifyErr := notify(alertMessage); notifyErr != nil {
					log.Printf("Notify error: %v", notifyErr)
				}
				// 2 second timeout is needed in MacOS for previous stock notification to get cleared.
				time.Sleep(2 * time.Second)
			}

		}
		log.Printf("Sleeping for 10 minutes")
		time.Sleep(10 * time.Minute)
	}

}
