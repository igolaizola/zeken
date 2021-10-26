# zeken

zeken is the translation of greedy or avaricious in euskera: https://hiztegiak.elhuyar.eus/eu/zeken

Binance trading bot based on input signals

## Requirements

### Binance

Right now only binance exchange is supported.

You need to obtain api key and api secret for your account. Here you have a guide providen by binance: https://www.binance.com/en/support/faq/360002502072

### Telegram bot

You need to follow these steps to create a bot account and obtain your telegram bot token.

 - Open a chat to https://t.me/BotFather
 - Press `/start` to start the conversation
 - Send `/newbot` to create a new bot
 - Choose a name and display name for your bot
 - Take note of your token
 - Open a chat to the bot that you have just created in order to have it on your chat history

### Telegram control chat id

This ID is the chat id of your telegram user. You can obtain it talking with the following telegram bot: https://t.me/username_to_id_bot

### Telegram signal chat id

This is the ID of the chat that will contain your trading signals. If the chat is public you can use https://t.me/username_to_id_bot to obtain the ID. If not, you must do the following:

 - Open telegram web: https://web.telegram.org/
 - Click on the private channel you want to get the ID
 - Check the URL on the browser, it will have a number similar to this: `511223344`
 - Add `-100` prefix to that number and the result will be your ID: `-100511223344` 

## Building source code

You need to install golang in order to build source code: https://golang.org/dl/

Build source code for multiple OS and architectures:

```
make app-build
```

Build docker images:

```
make docker-build
```

Build docker images using `docker buildx`:

```
make docker-buildx
```

## Running the bot

Main command to run the bot is `zeken run`.

You have three options to pass configuration parameters to your binary.

### Command line parameters

```
zeken run --exchange-key mykey --exchange-secret supersecret
```

### Config file

```
zeken run --config zeken.conf
```

Where config file looks like

```
exchange-key mykey
exchange-secret supersecret
```

### Environment variables

Environment variables must be upper case, prefixed with `ZEKEN` and use under scores.

```
export ZEKEN_EXCHANGE_KEY=mykey
export ZEKEN_EXCHANGE_SECRET=supersecret
```

## Deployment

This bot must be always running in order to run open trades and create new trades.
It is up to you how you want to deploy it. However, here you have some examples.

### Using a docker file

You can build the providen Dockerfile source

```
docker build -t zeken .
```

And run it using your current directory as volume to access your config file and/or database file.
```
docker run --rm -v $(pwd):/home zeken run --config zeken.conf
```

### Using systemd

You can create a systemd file `zeken.service` with all parameters included as environment variables and pointing to your binary.

```
[Unit]
Description=zeken bot
After=network.target
[Service]
Type=simple
User=johndoe
WorkingDirectory=/home/johndoe
Restart=always
RestartSec=5
Environment=ZEKEN_EXCHANGE_KEY=mykey
Environment=ZEKEN_EXCHANGE_SECRET=supersecret
ExecStart=/home/johndoe/bin/zeken run
[Install]
WantedBy=multi-user.target
```

Load and run the service

```
sudo cp zeken.service /etc/systemd/system
sudo systemctl daemon-reload
sudo systemctl start zeken
```

See the logs

```
journalctl -u zeken.service
```

Stop the service

```
sudo systemctl stop zeken
```
