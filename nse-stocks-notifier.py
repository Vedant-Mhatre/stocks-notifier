# Import the necessary modules
from nsepy import get_quote
import time
import os
import datetime
import logging
from json import load

FORMAT = '%(asctime)-15s- %(levelname)s - %(name)s -%(message)s'
logging.basicConfig(format=FORMAT, level=logging.DEBUG)
logger = logging.getLogger(__name__)

def check_ist_day_and_time():
  # Get the current day and time in IST
  current_day = datetime.datetime.now(datetime.timezone.utc).astimezone().strftime("%A")
  current_time = datetime.datetime.now(datetime.timezone.utc).astimezone().strftime("%H:%M")

  # Check if it's a weekday and the time is between 9:30am and 3pm, Indian stock market is active from 9:15 am to 3:30 pm, this script is designed to run only from 9:30 am to 3pm.
  if current_day in ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"] and "09:30" <= current_time <= "15:00":
    return True
  else:
    return False

# This function suspends execution of python script until stock market opens again.
def sleep_until_market_opens():
    # Get the current time
    now = datetime.datetime.now()
    
    # If it's Saturday or Sunday, it will calculate time until next monday's 9:30 am.
    if now.weekday() in [5, 6]: # Saturday or Sunday
        # Calculate the number of seconds until the next Monday at 9:30 am
        next_monday = (7 - now.weekday()) * 24 * 60 * 60 + 9 * 60 * 60 + 30 * 60
        # Sleep for the calculated number of seconds
        logger.info("Sleeping until next monday")
        time.sleep(next_monday)
        return

    # If it's a weekday, it will calculate time until next day's 9:30 am.
    # Set the target time to be 9:30 am
    target_time = now.replace(hour=9, minute=30, second=0, microsecond=0)

    # If the current time is after 3:00 pm, set the target time to be tomorrow
    if now.hour >= 15:
        target_time += datetime.timedelta(days=1)

    # Calculate the amount of time to sleep
    sleep_time = (target_time - now).total_seconds()
    # print(now, target_time, sleep_time)
    # Sleep
    logger.info("Sleeping until market opens at 9:30 am")
    time.sleep(sleep_time)

# A function to create a notification and play a sound
# Currently the notify function is desigined only for MacOS, in future it will support other OS as well.
def notify(title, text):
    os.system("""
              osascript -e 'display notification "{}" with title "{}"'
              """.format(text, title))
    os.system("afplay /System/Library/Sounds/Glass.aiff")

# A function to read data from stocks.json file which is stored in the same directory
# stocks.json file contains name of the stock and the value below which alert is to be triggered.
def read_json_data(filename):
    with open(filename, 'r') as f:
        data = load(f)
    return data

# A function to get real time stock price
# It uses nsepy python library to get real time price of a stock using get_quote.
def getStockPrice(stockName):
    try:
        # Get stock info for the given stock name
        stock_info = get_quote(stockName)

        # Extract the lastPrice field from the returned data
        # If the field is missing or has an invalid value, use a default value of 0
        stock_price = stock_info.get("data", [{}])[0].get("lastPrice", 0)

        # Convert the stock price to a floating-point number
        stock_price = float(stock_price.replace(",", ""))

        return stock_price
    except (KeyError, IndexError):
        # Handle the case where the stock name is invalid or the data does not have the expected format
        logger.critical(f"Invalid stock name or data format for {stockName}")
        return 0
    except Exception as e:
        # Handle any other exceptions that may occur
        logger.critical(f"Cannot get stock info for {stockName} because of exception: {e}")
        return 0

if __name__ == "__main__":
    while True:
        # Check if it is a weekday between 9:30am and 3pm in IST
        if check_ist_day_and_time():
            # Read the stocks data from the stocks.json file
            # This file can be updated without stopping the python process.
            stocks = {}
            try:
                stocks = read_json_data('stocks.json')
                logger.info(stocks)
            except Exception as e:
                # Log and notify if there is a problem with the stocks.json file
                logger.critical(f"There is problem with stocks.json: {e}")
                notify("Stock price alert", f"There is problem with your stocks.json, error: {e}")

            # Iterate through each stock in the stocks data
            for stock, price in stocks.items():
                try:
                    # Get the current stock price
                    stock_price = getStockPrice(stock)

                    # Print the stock price to the console
                    logger.info(f"{stock}: {stock_price}")

                    # If the stock price is invalid, create a notification
                    if stock_price == 0:
                        notify("Stock price alert", f"Error, couldn't find price of stock: {stock}")
                    # If the stock price is less than the alert price, create a notification
                    elif stock_price <= price:
                        notify("Stock price alert", f"{stock} stock price is less than {price}")
                except Exception as e:
                    # Handle any exceptions that may occur
                    logger.critical(f"Cannot get stock info for {stock} because of exception: {e}")
            # Sleep for 10 minutes before checking stock prices again
            time.sleep(600)
        else:
            # If it is not a weekday between 9:30am and 3pm in IST, log and sleep until the market opens
            logger.info("Market has closed")
            sleep_until_market_opens()
