<img width="250" src="https://user-images.githubusercontent.com/52707230/214772056-ef465e1e-4d71-47ec-9fb8-24a441e74e51.png" />

Stocks Notifier is a privacy-first alert tool that tracks real-time stock prices for a list of stocks and sends notifications when a price is lower than or equal to your threshold.

For docs, visit [stocksnotifier.com](https://stocksnotifier.com)

### Running locally

* Clone the repo.

* Copy `stocks.sample.json` to `stocks.json` and edit the value to set the lower threshold value at which you want to get alert.

* By default, this uses the public `stockprices.dev` API for real-time US equities and ETFs. It expects plain US tickers (e.g., `AAPL`, `TSLA`). If your symbol has a suffix (like `.NS`), it will be ignored for real-time lookups.
* If you want delayed data as a fallback, set `STOCKS_NOTIFIER_ALLOW_DELAYED=1`. This uses Stooq (daily close) when real-time fails.

* Run go file ` go run stocks-notifier.go . `
Pass the directory where stocks.json is located.

* If you want to run this file in background:
``` nohup go run stocks-notifier.go . & ``` this will output logs to file named nohup.out
