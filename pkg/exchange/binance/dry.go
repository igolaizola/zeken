package binance

import (
	"context"
	"fmt"
	"strings"

	"github.com/igolaizola/zeken/pkg/exchange"
	"github.com/shopspring/decimal"
)

type binanceExchangeDry struct {
	exchange.Exchange
}

func NewDry(log func(v ...interface{}), debug bool) exchange.Exchange {
	ex := New(log, "", "", debug)
	return &binanceExchangeDry{
		Exchange: ex,
	}
}

func (e *binanceExchangeDry) Buy(ctx context.Context, symbol string, quoteQuantity, price decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	price, err := e.Price(ctx, symbol)
	if err != nil {
		return zero, zero, err
	}
	qty := quoteQuantity.Div(price).Round(4)
	return quoteQuantity, qty, nil
}

func (e *binanceExchangeDry) Sell(ctx context.Context, symbol string, quantity decimal.Decimal) (decimal.Decimal, error) {
	price, err := e.Price(ctx, symbol)
	if err != nil {
		return zero, err
	}
	quoteQty := quantity.Mul(price).Round(4)
	return quoteQty, nil
}

func (e *binanceExchangeDry) CreateStopLimit(ctx context.Context, symbol string, quantity, target, stoploss decimal.Decimal) (string, []string, error) {
	id1 := fmt.Sprintf("greater_%s_%s", target, quantity)
	id2 := fmt.Sprintf("less_%s_%s", stoploss, quantity)
	return "", []string{id1, id2}, nil
}

func (e *binanceExchangeDry) CancelStopLimit(ctx context.Context, symbol string, id string) error {
	return nil
}

func (e *binanceExchangeDry) Status(ctx context.Context, symbol string, id string) (bool, decimal.Decimal, error) {
	price, err := e.Price(ctx, symbol)
	if err != nil {
		return false, zero, err
	}
	split := strings.Split(id, "_")
	if len(split) != 3 {
		return false, zero, fmt.Errorf("binance: invalid dry order id: %s", id)
	}
	var compare func(decimal.Decimal) bool
	switch split[0] {
	case "greater":
		compare = price.GreaterThan
	case "less":
		compare = price.LessThan
	default:
		return false, zero, fmt.Errorf("binance: invalid dry order id: %s", id)
	}
	target, err := decimal.NewFromString(split[1])
	if err != nil {
		return false, zero, fmt.Errorf("binance: invalid dry order id %s: %w", id, err)
	}
	quantity, err := decimal.NewFromString(split[2])
	if err != nil {
		return false, zero, fmt.Errorf("binance: invalid dry order id %s: %w", id, err)
	}
	quoteQty := quantity.Mul(price)
	return compare(target), quoteQty, nil
}

func (e *binanceExchangeDry) Balance(ctx context.Context, currency string) (decimal.Decimal, error) {
	return decimal.NewFromFloat(100.0), nil
}
