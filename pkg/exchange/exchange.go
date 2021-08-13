package exchange

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
)

type Exchange interface {
	Buy(ctx context.Context, symbol string, quoteQuantity, price decimal.Decimal) (quoteQty decimal.Decimal, qty decimal.Decimal, err error)
	Sell(ctx context.Context, symbol string, quantity decimal.Decimal) (quoteQty decimal.Decimal, err error)
	CreateStopLimit(ctx context.Context, symbol string, quantity, target, stoploss decimal.Decimal) (string, []string, error)
	CancelStopLimit(ctx context.Context, symbol string, id string) error
	Status(ctx context.Context, symbol string, id string) (completed bool, quoteQty decimal.Decimal, err error)
	Price(ctx context.Context, symbol string) (decimal.Decimal, error)
	Balance(ctx context.Context, currency string) (decimal.Decimal, error)
	Symbol(base, quote string) string
}

var ErrOrderCanceled = errors.New("order canceled")
