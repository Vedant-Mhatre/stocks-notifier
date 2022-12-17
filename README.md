# NSE Stocks Notifier

## About
This script tracks real-time stock prices for a list of NSE stocks and sends notifications when certain criteria are met. This is designed to run from 9:30 am to 3pm on all weekdays and it automatically sleeps on all other times. It check price every 10 minutes and alerts if price of stock mentioned in [stocks.json](https://github.com/Vedant-Mhatre/nse-stocks-notifier/blob/main/stocks.json) is less than or equal to real time value of that stock.

Almost all existing softwares which can give this notifications require you to sign up using your phone number of email and also take up a lot of CPU and memory. This simple script does not require anyone to signup and is also light on resources.

### Requirements
* python3
* nsepy

### Running locally

* Clone the repo.

* Install nsepy using pip3 ``` pip3 install nsepy ```

* Edit [stocks.json](https://github.com/Vedant-Mhatre/nse-stocks-notifier/blob/main/stocks.json) according to your needs. Enter the stock name according to [NSE](https://www1.nseindia.com/live_market/dynaContent/live_watch/equities_stock_watch.htm) and threshold value you want to set to get alert.

* Run python file
``` python3 nse-stocks-notifier.py ```

* If you want to run this file in background: 
``` nohup python3 nse-stocks-notifier.py & ``` this will output logs to file named nohup.out

### Limitations
* Only designed to work on MacOS, support for other OS will be added soon.
* This script will only give notifications from 9:30 am to 3 pm on weekdays, in future will be adding option for user to set the starting and end time for which the user wants notifications.
* This script runs on all weekdays, also on holidays when markets is closed. In future support will be added such that it will only run on days when NSE market is open.

### To-Do list:
* Replace nsepy library with a better and more reliable option.
* Only works on MacOS, add support for other OS.
