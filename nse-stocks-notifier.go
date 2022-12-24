package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/piquette/finance-go/quote"
)

func readJSONData(filename string) (map[string]interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist")
		}
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	var data map[string]interface{}
	if err := decoder.Decode(&data); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("empty file")
		}
		return nil, fmt.Errorf("invalid JSON data: %v", err)
	}

	return data, nil
}

func notify(title, text string) error {
	command := "osascript"
	arg1 := "-e"
	arg2 := `display notification "` + text + `" with title "` + title + `"`
	cmd := exec.Command(command, arg1, arg2)
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("afplay", "/System/Library/Sounds/Glass.aiff")
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func handleError(err error) {
	fmt.Printf("Error: %v\n", err)
	notify("Stock price alert", fmt.Sprintf("Error: %v", err))
}

func GetStockPrice(symbol string) (float64, error) {
	if symbol == "" {
		return 0, fmt.Errorf("symbol cannot be empty")
	}

	if !isValidSymbol(symbol) {
		return 0, fmt.Errorf("invalid symbol %q", symbol)
	}

	stockQuote, err := quote.Get(symbol)
	if err != nil {
		if isNetworkError(err) {
			return 0, fmt.Errorf("failed to get stock quote due to network error: %v", err)
		}

		handleError(err)
		return 0, fmt.Errorf("failed to get stock quote for symbol %q: %v", symbol, err)
	}

	if stockQuote == nil {
		return 0, fmt.Errorf("received nil stock quote for symbol %q", symbol)
	}

	return stockQuote.RegularMarketPrice, nil
}

func isValidSymbol(symbol string) bool {
	// Add code to check if the symbol is a valid length and contains only alphanumeric characters
	// Code will be added later
	return true
}

func isNetworkError(err error) bool {
	// Add code to check if the error is a network error
	// Code will be added later
	return false
}

func main() {

	var stocks map[string]interface{}
	stocks, err := readJSONData("stocks.json")
	if err != nil {
		handleError(err)
	}

	for symbol := range stocks {
		price, err := GetStockPrice(symbol + ".NS")
		if err != nil {
			handleError(err)
			continue
		}

		// notify("Stock price alert", fmt.Sprintf("Price of stock %v: %.2f", symbol, price))
		fmt.Printf("Price of stock %q: %.2f\n", symbol, price)
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Printf("Total Allocated = %v MiB", mem.TotalAlloc/1024/1024)
	fmt.Printf("\tSys = %v MiB", mem.Sys/1024/1024)
	fmt.Printf("\tHeapAlloc = %v MiB", mem.HeapAlloc/1024/1024)
	fmt.Printf("\tHeapSys = %v MiB", mem.HeapSys/1024/1024)
}
