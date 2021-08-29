package zeken

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/igolaizola/zeken/pkg/exchange"
	"github.com/igolaizola/zeken/pkg/exchange/binance"
	"github.com/igolaizola/zeken/pkg/signal"
	"github.com/igolaizola/zeken/pkg/telegram"
	"github.com/igolaizola/zeken/pkg/trade"
	"github.com/igolaizola/zeken/pkg/trade/bolt"
	"github.com/shopspring/decimal"
)

var version = "v210829a"

type Bot struct {
	run          func(context.Context) error
	ctx          context.Context
	cancel       context.CancelFunc
	exchange     exchange.Exchange
	log          func(v ...interface{})
	parser       signal.Parser
	maxTrades    int
	balanceRatio float64
	trades       map[string]*trade.Trader
	lock         sync.Mutex
	store        trade.Store
	currency     string
	dry          bool
}

func NewBot(dbPath, apiKey, apiSecret, token string, controlChatID, signalChatID, maxTrades int, balanceRatio float64, currency string, dry, debug bool) (*Bot, error) {
	tgbot, err := telegram.New(token, controlChatID)
	if err != nil {
		return nil, fmt.Errorf("zeken: couldn't create telegram bot: %w", err)
	}
	log := tgbot.Print
	var ex exchange.Exchange
	if dry {
		ex = binance.NewDry(log, debug)
	} else {
		ex = binance.New(log, apiKey, apiSecret, debug)
	}

	parser, err := signal.NewParser()
	if err != nil {
		return nil, fmt.Errorf("zeken: couldn't create parser: %w", err)
	}
	store, err := bolt.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("zeken: couldn't create db: %w", err)
	}
	b := &Bot{
		ctx:          context.TODO(),
		run:          tgbot.Run,
		log:          log,
		exchange:     ex,
		parser:       parser,
		maxTrades:    maxTrades,
		balanceRatio: balanceRatio,
		trades:       make(map[string]*trade.Trader),
		lock:         sync.Mutex{},
		store:        store,
		currency:     currency,
		dry:          dry,
	}
	tgbot.HandleChat(int64(signalChatID), true, func(msg string) {
		b.handle(msg)
	})
	tgbot.HandleCommand("status", func(_ string) {
		if len(b.trades) == 0 {
			b.log("no trades running")
			return
		}
		for _, t := range b.trades {
			profit, perc, elapsed := t.Status()
			b.log(fmt.Sprintf("📈 %s %s%% %s %s %s", t.Base, perc.Mul(decimal.NewFromInt(100)).StringFixed(2), profit.StringFixed(2), t.Quote, elapsed.Round(time.Second)))
		}
	})
	tgbot.HandleCommand("sell", func(msg string) {
		t, ok := b.trades[msg]
		if !ok {
			b.log(fmt.Sprintf("trade %s not found", msg))
			return
		}
		b.log(fmt.Sprintf("selling %s", msg))
		t.Sell()
	})
	tgbot.HandleCommand("shutdown", func(_ string) {
		b.log("shutting down")
		b.shutdown()
	})
	return b, nil
}

func (b *Bot) Run(ctx context.Context) error {
	b.ctx, b.cancel = context.WithCancel(ctx)
	b.log(fmt.Sprintf("🤖 zeken bot running\n- version: %s\n- dry mode: %t", version, b.dry))
	defer b.log("🛑 zeken bot stopped")
	if err := b.resume(); err != nil {
		b.log(err)
	}
	return b.run(b.ctx)
}

func (b *Bot) handle(text string) {
	sig, err := b.parser.Parse(text)
	if err != nil {
		b.log(err)
		return
	}
	if err := b.signal(sig); err != nil {
		b.log(err)
		return
	}
}

func (b *Bot) resume() error {
	to := time.Now().UTC().Add(24 * time.Hour)
	from := time.Now().UTC().Add(-365 * 24 * time.Hour)
	trades, err := b.store.List(from, to, false)
	if err != nil {
		return fmt.Errorf("zeken: couldn't get trades from db: %w", err)
	}
	for _, tr := range trades {
		tr := tr
		trader := trade.NewTrader(b.log, b.exchange, tr, 5*time.Second)
		b.lock.Lock()
		b.trades[tr.Base] = trader
		b.lock.Unlock()
		go func() {
			b.trade(trader, false)
		}()
	}
	return nil
}

func (b *Bot) signal(sig *signal.Signal) error {
	if sig.Quote != b.currency {
		return fmt.Errorf("zeken: quote currency %s not supported", sig.Quote)
	}
	var found bool
	for _, e := range sig.Exchanges {
		if e == "BINANCE" {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("zeken: exchange %v not supported", sig.Exchanges)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	if len(b.trades) >= b.maxTrades {
		return fmt.Errorf("maximum number of trades running: %d", len(b.trades))
	}
	if _, ok := b.trades[sig.Base]; ok {
		return fmt.Errorf("there is already a running trade for %s", sig.Base)
	}

	// Calculate quote quantity based on open trades, available balance and balance ratio
	quoteQty := decimal.NewFromFloat(10.0)
	if !b.dry {
		openTradesQty := decimal.Zero
		for _, t := range b.trades {
			openTradesQty = openTradesQty.Add(t.QuoteQuantity)
		}

		available, err := b.exchange.Balance(b.ctx, sig.Quote)
		if err != nil {
			return fmt.Errorf("couldn't get balance: %w", err)
		}
		quoteQty = openTradesQty.Add(available).Mul(decimal.NewFromFloat(b.balanceRatio)).Div(decimal.NewFromInt(int64(b.maxTrades)))
		if quoteQty.GreaterThanOrEqual(available) {
			quoteQty = available.Mul(decimal.NewFromFloat(0.99))
		}
	}

	tr := trade.New(sig.Base, sig.Quote, sig.Start, sig.Targets, sig.Stop, quoteQty)
	trader := trade.NewTrader(b.log, b.exchange, tr, 5*time.Second)
	b.trades[tr.Base] = trader

	go func() {
		b.trade(trader, true)
	}()
	return nil
}

func (b *Bot) trade(t *trade.Trader, new bool) {
	defer func() {
		b.lock.Lock()
		delete(b.trades, t.Base)
		b.lock.Unlock()
	}()
	b.log(fmt.Sprintf("⚙️ running trade %s", t.Base))
	if new {
		if err := t.Create(b.ctx); err != nil {
			b.log(err)
			return
		}
		if err := b.store.Update(t.Trade); err != nil {
			b.log(fmt.Errorf("zeken: couldn't update trade: %w", err))
		}
	}
	err := t.Run(b.ctx)
	if err != nil {
		b.log(err)
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	if errors.Is(err, exchange.ErrOrderCanceled) {
		b.log(fmt.Sprintf("⚠️ %s finished with error because order was canceled, it will be removed from database", t.Base))
		if err := b.store.Delete(t.Trade); err != nil {
			b.log(fmt.Errorf("zeken: couldn't delete trade: %w", err))
		}
		return
	}

	if err := b.store.Update(t.Trade); err != nil {
		b.log(fmt.Errorf("zeken: couldn't update trade: %w", err))
	}
	profit, perc, elapsed := t.Status()
	emoji := "💰"
	if profit.LessThan(decimal.Zero) {
		emoji = "❌"
	}
	b.log(emoji, fmt.Sprintf("finished %s %s%% %s %s %s", t.Base, perc.Mul(decimal.NewFromInt(100)).StringFixed(2), profit.StringFixed(2), t.Quote, elapsed.Round(time.Second)))
}

func (b *Bot) shutdown() {
	b.cancel()
}
