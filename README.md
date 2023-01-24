# NSE Stocks Notifier

## Update

It was observed that python was little bit slow and was also using a lot of memory, hence this project is being migrated to golang.

## About

This script tracks real-time stock prices for a list of NSE stocks and sends notifications when certain criteria are met. This is designed to run from 9:30 am to 3pm on all weekdays and it automatically sleeps on all other times. It check price every 10 minutes and alerts if price of stock mentioned in [stocks.json](https://github.com/Vedant-Mhatre/nse-stocks-notifier/blob/main/stocks.json) is less than or equal to real time value of that stock.

Almost all existing softwares which can give this notifications require you to sign up using your phone number of email and also take up a lot of CPU and memory. This simple script does not require anyone to signup and is also light on resources.

### Requirements

* golang

### Running locally

* Clone the repo.

* Copy `stocks.sample.json` to `stocks.json` and edit the value to set the lower threshold value at which you want to get alert.

* Run go file ` go run . . `
Here second . represents the directory where stocks.json is stored, '.' if you are running this inside same directory.

* If you want to run this file in background:
``` nohup go run . . & ``` this will output logs to file named nohup.out

### Limitations

* Only designed to work on MacOS, support for other OS will be added soon.
* This script will only give notifications from 9:30 am to 3 pm on weekdays, in future will be adding option for user to set the starting and end time for which the user wants notifications.
* This script runs on all weekdays, also on holidays when markets is closed. In future support will be added such that it will only run on days when NSE market is open.

### To-Do list

* Only works on MacOS, add support for other OS.
* Add support for all stocks and cryptos and not just NSE.
