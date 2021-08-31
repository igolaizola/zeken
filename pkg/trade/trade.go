package trade

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/igolaizola/zeken/pkg/exchange"
	"github.com/shopspring/decimal"
)

type Trade struct {
	StartTime        time.Time
	Base             string
	Quote            string
	StartPrice       decimal.Decimal
	Targets          []decimal.Decimal
	StopPrice        decimal.Decimal
	QuoteQuantity    decimal.Decimal
	Quantity         decimal.Decimal
	EndQuoteQuantity decimal.Decimal
	EndTime          time.Time
	// TODO(igolaizola): change implementation to have a single ID
	OrderIDs      []string
	OrderListID   string
	CurrentTarget int
}

func New(base, quote string, start decimal.Decimal, targets []decimal.Decimal, stop, quoteQuantity decimal.Decimal) *Trade {
	return &Trade{
		StartTime:     time.Now().UTC(),
		Base:          base,
		Quote:         quote,
		StartPrice:    start,
		Targets:       targets,
		StopPrice:     stop,
		QuoteQuantity: quoteQuantity,
	}
}

type Trader struct {
	*Trade
	lastPrice decimal.Decimal
	symbol    string
	log       func(v ...interface{})
	exchange  exchange.Exchange
	sell      chan struct{}
	wait      time.Duration
	update    func(t *Trade) error
}

func NewTrader(log func(v ...interface{}), ex exchange.Exchange, t *Trade, wait time.Duration, update func(t *Trade) error) *Trader {
	return &Trader{
		Trade:    t,
		symbol:   ex.Symbol(t.Base, t.Quote),
		log:      log,
		exchange: ex,
		sell:     make(chan struct{}),
		wait:     wait,
		update:   update,
	}
}

func (t *Trader) Create(ctx context.Context) error {
	if err := t.buy(ctx); err != nil {
		return err
	}
	lower := t.StopPrice
	upper := t.Targets[len(t.Targets)-1]
	if err := t.createStopLimit(ctx, upper, lower); err != nil {
		defer t.log(fmt.Sprintf("⚠️ Warning! %s has been bought, but order creation failed. You must sell it manually", t.symbol))
		return fmt.Errorf("trade: couldn't create order for %s: %w", t.symbol, err)
	}
	if err := t.update(t.Trade); err != nil {
		t.log("trade: couldn't update %s: %w", t.Base, err)
	}
	return nil
}

func (t *Trader) Run(ctx context.Context) error {
	lower := t.StopPrice
	upper := t.Targets[len(t.Targets)-1]
	previous := t.StartPrice
	target := t.Targets[t.CurrentTarget]

	// Don't wait ticker time on first run
	closedTick := make(chan time.Time)
	close(closedTick)
	tick := (<-chan time.Time)(closedTick)
	ticker := time.NewTicker(t.wait)

	var canceled bool
	var forceSell bool
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick:
		case <-t.sell:
			forceSell = true
		}
		tick = ticker.C

		// Check if orders have been completed
		for _, id := range t.OrderIDs {
			ok, endQuoteQty, err := t.status(ctx, id)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			t.EndQuoteQuantity = endQuoteQty
			t.EndTime = time.Now().UTC()
			if err := t.update(t.Trade); err != nil {
				t.log("trade: couldn't update %s: %w", t.Base, err)
			}
			return nil
		}

		// Check price
		price, err := t.exchange.Price(ctx, t.symbol)
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			continue
		}
		if err != nil {
			t.log(fmt.Sprintf("%v (%T)", err, err))
			continue
		}
		t.lastPrice = price

		// Force sell if price is lower than stoploss * 0.99 or greater than upper * 1.01
		if forceSell || price.LessThan(lower.Mul(decimal.NewFromFloat(0.99))) || price.GreaterThan(upper.Mul(decimal.NewFromFloat(1.01))) {
			if !canceled {
				if err := t.cancelStopLimit(ctx); err != nil {
					t.log(err)
					continue
				}
			}
			canceled = true
			if err := t.forceSell(ctx); err != nil {
				t.log(err)
				continue
			}
			if err := t.update(t.Trade); err != nil {
				t.log("trade: couldn't update %s: %w", t.Base, err)
			}
			return nil
		}

		// Target not reached
		if price.LessThan(target) {
			continue
		}

		// This was the last target
		if t.CurrentTarget == len(t.Targets)-1 {
			continue
		}

		// Move to next target
		if err := t.cancelStopLimit(ctx); err != nil {
			return err
		}
		t.CurrentTarget++
		lower = previous
		previous = target
		target = t.Targets[t.CurrentTarget]
		t.log(fmt.Sprintf("✔️ %s reached target %d", t.Base, t.CurrentTarget))
		if err := t.createStopLimit(ctx, upper, lower); err != nil {
			return err
		}
		if err := t.update(t.Trade); err != nil {
			t.log("trade: couldn't update %s: %w", t.Base, err)
		}
		canceled = false
	}
}

func (t *Trader) Status() (decimal.Decimal, decimal.Decimal, time.Duration) {
	currentQuoteQuantity := t.lastPrice.Mul(t.Quantity)
	profit := currentQuoteQuantity.Sub(t.QuoteQuantity)
	percentage := profit.Div(t.QuoteQuantity)
	elapsed := time.Since(t.StartTime)
	return profit, percentage, elapsed
}

func (t *Trader) Sell() {
	close(t.sell)
}

func (t *Trader) status(ctx context.Context, id string) (bool, decimal.Decimal, error) {
	for {
		select {
		case <-ctx.Done():
		default:
		}
		ok, endQuoteQty, err := t.exchange.Status(ctx, t.symbol, id)
		var netErr net.Error
		if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
			continue
		}
		if err != nil {
			return false, decimal.Decimal{}, fmt.Errorf("trade: couldn't get order status: %w (%T)", err, err)
		}
		return ok, endQuoteQty, nil
	}
}

func (t *Trader) cancelStopLimit(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
		default:
		}
		err := t.exchange.CancelStopLimit(ctx, t.symbol, t.OrderListID)
		var netErr net.Error
		if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
			continue
		}
		if err != nil {
			return fmt.Errorf("trade: couldn't delete order list %s:  %w (%T)", t.OrderListID, err, err)
		}
		return nil
	}
}

func (t *Trader) createStopLimit(ctx context.Context, upper, lower decimal.Decimal) error {
	for {
		select {
		case <-ctx.Done():
		default:
		}
		var err error
		t.OrderListID, t.OrderIDs, err = t.exchange.CreateStopLimit(ctx, t.symbol, t.Quantity, upper, lower)
		var netErr net.Error
		if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
			continue
		}
		if err != nil {
			return fmt.Errorf("trade: couldn't create order for %s: %w (%T)", t.symbol, err, err)
		}
		return nil
	}
}

func (t *Trader) buy(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
		default:
		}
		quoteQty, qty, err := t.exchange.Buy(ctx, t.symbol, t.QuoteQuantity, t.StartPrice)
		var netErr net.Error
		if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
			continue
		}
		if err != nil {
			return fmt.Errorf("trade: couldn't buy %s at price %s: %w (%T)", t.Base, t.StartPrice, err, err)
		}
		t.QuoteQuantity = quoteQty
		t.Quantity = qty
		return nil
	}
}

func (t *Trader) forceSell(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
		default:
		}
		quoteQty, err := t.exchange.Sell(ctx, t.symbol, t.Quantity)
		var netErr net.Error
		if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
			continue
		}
		if err != nil {
			return fmt.Errorf("trade: couldn't force sell %s: %w (%T)", t.Base, err, err)
		}
		t.EndQuoteQuantity = quoteQty
		t.EndTime = time.Now().UTC()
		return nil
	}
}
