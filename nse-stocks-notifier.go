package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/piquette/finance-go/quote"
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
	fmt.Println("\nTo run the program, use the following command:")
	fmt.Printf("\n%s <directory>\n", os.Args[0])
	fmt.Println("\nIf the stocks.json file is in the current directory, use '.' as the directory path.")
	os.Exit(1)
}

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

	log.Print(data)
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

func notifyError(err error) {
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

		notifyError(err)
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

	dir, error := getDirectoryPath()
	if error != nil {
		fmt.Println(error)
		directoryPathHelpMessage()
	}

	for {
		var stocks map[string]interface{}
		stocks, err := readJSONData(dir + "/stocks.json")
		if err != nil {
			notifyError(err)
		}

		for symbol := range stocks {
			price, err := GetStockPrice(symbol + ".NS")
			if err != nil {
				notifyError(err)
				continue
			}

			// notify("Stock price alert", fmt.Sprintf("Price of stock %v: %.2f", symbol, price))
			log.Printf("Price of stock %q: %.2f\n", symbol, price)
		}
		time.Sleep(10 * time.Minute)
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Printf("Total Allocated = %v MiB", mem.TotalAlloc/1024/1024)
	fmt.Printf("\tSys = %v MiB", mem.Sys/1024/1024)
	fmt.Printf("\tHeapAlloc = %v MiB", mem.HeapAlloc/1024/1024)
	fmt.Printf("\tHeapSys = %v MiB", mem.HeapSys/1024/1024)
}
