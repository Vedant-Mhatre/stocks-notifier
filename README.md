<img width="250" src="https://user-images.githubusercontent.com/52707230/214772056-ef465e1e-4d71-47ec-9fb8-24a441e74e51.png" />

Stocks Notifier is a software which tracks real-time stock prices for a list of stocks and sends notifications when price of stock is lower than or equal to threshold value.

Why? Almost all existing softwares require you to sign up using your phone number or email and also take up a lot of CPU and memory. This notifier does not require anyone to signup and is also light on resources.

### Running locally

* Clone the repo.

* Copy `stocks.sample.json` to `stocks.json` and edit the value to set the lower threshold value at which you want to get alert.

* Check [Yahoo Finance](https://finance.yahoo.com/lookup) page for naming convention, eg: for NSE stocks .NS has to be added after stock name.

* Run go file ` go run stocks-notifier.go . `
Pass the directory where stocks.json is located.

* If you want to run this file in background:
``` nohup go run stocks-notifier.go . & ``` this will output logs to file named nohup.out

### Tested on:

- [x] MacOS
- [x] Windows
- [x] Ubuntu 18
