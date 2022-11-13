package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/igolaizola/zeken"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	// Create signal based context
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
			cancel()
		}
		signal.Stop(c)
	}()

	// Launch command
	cmd := newCommand()
	if err := cmd.ParseAndRun(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func newCommand() *ffcli.Command {
	fs := flag.NewFlagSet("zeken", flag.ExitOnError)

	return &ffcli.Command{
		ShortUsage: "zeken [flags] <subcommand>",
		FlagSet:    fs,
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
		Subcommands: []*ffcli.Command{
			newServeCommand(),
		},
	}
}

func newServeCommand() *ffcli.Command {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	_ = fs.String("config", "", "config file (optional)")

	db := fs.String("db", "zeken.db", "database path")
	parser := fs.String("parser", "json", "signal parser name")
	key := fs.String("exchange-key", "", "binance api key")
	secret := fs.String("exchange-secret", "", "binance api secret")
	proxy := fs.String("proxy", "", "proxy to be used on requests to binance")
	token := fs.String("telegram-token", "", "telegram token")
	controlChat := fs.Int("telegram-control-chat", 0, "telegram chat id for logs and commands")
	signalChat := fs.Int("telegram-signal-chat", 0, "telegram chat id to read signals")
	mtID := fs.Int("mt-id", 0, "mtproto app id")
	mtHash := fs.String("mt-hash", "", "mtproto app hash")
	mtPhone := fs.String("mt-phone", "", "mtproto phone number")
	mtSession := fs.String("mt-session", "", "mtproto session file")
	mtChat := fs.Int64("mt-chat", 0, "mtproto chat id to read signals")
	maxTrades := fs.Int("max-trades", 5, "max simultaneus trades")
	maxTarget := fs.Int("max-target", 5, "max target to sell")
	balance := fs.Float64("balance-ratio", 0.99, "balance ratio to be used")
	currency := fs.String("currency", "USDT", "quote currency")
	dry := fs.Bool("dry", false, "enable dry mode")
	debug := fs.Bool("debug", false, "enable debug mode")

	return &ffcli.Command{
		Name:       "run",
		ShortUsage: "zeken run [flags] <key> <value data...>",
		Options: []ff.Option{
			ff.WithConfigFileFlag("config"),
			ff.WithConfigFileParser(ff.PlainParser),
			ff.WithEnvVarPrefix("ZEKEN"),
		},
		ShortHelp: "run zeken bot",
		FlagSet:   fs,
		Exec: func(ctx context.Context, args []string) error {
			if *db == "" {
				return errors.New("missing db path")
			}
			if *dry && !strings.HasSuffix(*db, ".dry.db") {
				*db = fmt.Sprintf("%s.dry.db", strings.TrimSuffix(*db, ".db"))
			}
			if !*dry {
				if *key == "" {
					return errors.New("missing exchange api key")
				}
				if *secret == "" {
					return errors.New("missing exchange api secret")
				}
			}
			if *token == "" {
				return errors.New("missing telegram token")
			}
			if *controlChat == 0 {
				return errors.New("missing telegram control chat")
			}
			if *signalChat == 0 {
				return errors.New("missing telegram signal chat")
			}
			if *currency == "" {
				return errors.New("missing currency")
			}
			if *mtID > 0 || *mtHash != "" || *mtPhone != "" || *mtSession != "" || *mtChat > 0 {
				if *mtID == 0 {
					return errors.New("missing mtproto app id")
				}
				if *mtHash == "" {
					return errors.New("missing mtproto app hash")
				}
				if *mtPhone == "" {
					return errors.New("missing mtproto phone number")
				}
				if *mtSession == "" {
					return errors.New("missing mtproto session file")
				}
				if *mtChat == 0 {
					return errors.New("missing mtproto chat")
				}
			}
			bot, err := zeken.NewBot(*db, *key, *secret, *proxy, *token, *parser, *controlChat, *signalChat, *mtID, *mtHash, *mtPhone, *mtSession, *mtChat, *maxTrades, *maxTarget, *balance, *currency, *dry, *debug)
			if err != nil {
				return err
			}
			return bot.Run(ctx)

		},
	}
}
