# NSE Stocks Notifier

## About

This script tracks real-time stock prices for a list of NSE stocks and sends notifications when certain criteria are met. It check price every 10 minutes and alerts if price of stock mentioned in [stocks.json](https://github.com/Vedant-Mhatre/nse-stocks-notifier/blob/main/stocks.json) is less than or equal to real time value of that stock.

Almost all existing softwares which can give this notifications require you to sign up using your phone number of email and also take up a lot of CPU and memory. This simple script does not require anyone to signup and is also light on resources.

### Requirements

* golang

### Running locally

* Clone the repo.

* Copy `stocks.sample.json` to `stocks.json` and edit the value to set the lower threshold value at which you want to get alert.

* For NSE stocks .NS has to be added after stock name, check [Yahoo Finance](https://finance.yahoo.com/lookup) page for naming convention.

* Run go file ` go run . . `
Here second . represents the directory where stocks.json is stored, '.' if you are running this inside same directory.

* If you want to run this file in background:
``` nohup go run . . & ``` this will output logs to file named nohup.out

### To-Do list

- [ ] Only works on MacOS, add support for other OS.
- [x] Add support for all stocks and cryptos and not just NSE.
