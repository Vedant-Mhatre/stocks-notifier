<img width="250" src="https://user-images.githubusercontent.com/52707230/214772056-ef465e1e-4d71-47ec-9fb8-24a441e74e51.png" />

Stocks Notifier is a software which tracks real-time stock prices for a list of stocks and sends notifications when price of stock is lower than or equal to threshold value.

For docs, visit [blog.vmhatre.com/stocks-notifier/](https://blog.vmhatre.com/stocks-notifier/)

### Running locally

* Clone the repo.

* Copy `stocks.sample.json` to `stocks.json` and edit the value to set the lower threshold value at which you want to get alert.

* Check [Yahoo Finance](https://finance.yahoo.com/lookup) page for naming convention, eg: for NSE stocks .NS has to be added after stock name.

* Run go file ` go run stocks-notifier.go . `
Pass the directory where stocks.json is located.

* If you want to run this file in background:
``` nohup go run stocks-notifier.go . & ``` this will output logs to file named nohup.out
