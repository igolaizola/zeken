package trade

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestOrderCompleted(t *testing.T) {
	targets := []decimal.Decimal{
		decimal.NewFromFloat(11.0),
		decimal.NewFromFloat(12.0),
		decimal.NewFromFloat(13.0),
		decimal.NewFromFloat(14.0),
		decimal.NewFromFloat(15.0),
	}
	startPrice := decimal.NewFromFloat(10.0)
	quoteQty := decimal.NewFromFloat(100.0)
	tr := New("IGO", "USDT", startPrice, targets, decimal.NewFromFloat(9.0), quoteQty)

	ex := &mockExchange{
		price: decimal.NewFromFloat(10.1),
		inc:   decimal.NewFromFloat(1.0),
	}

	trader := NewTrader(log.Println, ex, tr, 5, 10*time.Millisecond, func(t *Trade) error { return nil })
	if err := trader.Create(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := trader.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := tr.EndQuoteQuantity
	want := decimal.NewFromFloat(151.0)
	if !got.Equal(want) {
		t.Errorf("wrong end quote quantity: want %s, got %s", want, got)
	}
	if ex.created != 5 {
		t.Errorf("wrong number of created orders: want 5, got %d", ex.canceled)
	}
	if ex.canceled != 4 {
		t.Errorf("wrong number of canceled orders: want 4, got %d", ex.canceled)
	}
}

func TestForceSell(t *testing.T) {
	targets := []decimal.Decimal{
		decimal.NewFromFloat(11.0),
		decimal.NewFromFloat(12.0),
		decimal.NewFromFloat(13.0),
		decimal.NewFromFloat(14.0),
		decimal.NewFromFloat(15.0),
	}
	startPrice := decimal.NewFromFloat(10.0)
	quoteQty := decimal.NewFromFloat(100.0)
	tr := New("IGO", "USDT", startPrice, targets, decimal.NewFromFloat(9.0), quoteQty)

	ex := &mockExchange{
		forceSell: true,
		price:     decimal.NewFromFloat(10.1),
		inc:       decimal.NewFromFloat(1.0),
	}

	trader := NewTrader(log.Println, ex, tr, 5, 10*time.Millisecond, func(t *Trade) error { return nil })
	if err := trader.Create(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := trader.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := tr.EndQuoteQuantity
	want := decimal.NewFromFloat(161.0)
	if !got.Equal(want) {
		t.Errorf("wrong end quote quantity: want %s, got %s", want, got)
	}
	if ex.sold != true {
		t.Errorf("force sell hasn't been called")
	}
}

type mockExchange struct {
	forceSell bool
	price     decimal.Decimal
	inc       decimal.Decimal
	target    decimal.Decimal
	quantity  decimal.Decimal
	created   int
	canceled  int
	sold      bool
}

func (e *mockExchange) Buy(ctx context.Context, symbol string, quoteQuantity, price decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	return quoteQuantity, quoteQuantity.Div(price), nil
}
func (e *mockExchange) Sell(ctx context.Context, symbol string, quantity decimal.Decimal) (decimal.Decimal, error) {
	e.sold = true
	return quantity.Mul(e.price), nil
}
func (e *mockExchange) CreateStopLimit(ctx context.Context, symbol string, quantity, target, stoploss decimal.Decimal) (string, []string, error) {
	e.target = target
	e.quantity = quantity
	e.created++
	return "", []string{""}, nil
}
func (e *mockExchange) CancelStopLimit(ctx context.Context, symbol string, id string) error {
	e.canceled++
	return nil
}
func (e *mockExchange) Status(ctx context.Context, symbol string, id string) (bool, decimal.Decimal, error) {
	if e.forceSell {
		return false, decimal.Decimal{}, nil
	}
	ok := e.price.GreaterThan(e.target)
	return ok, e.quantity.Mul(e.price), nil
}
func (e *mockExchange) Price(ctx context.Context, symbol string) (decimal.Decimal, error) {
	e.price = e.price.Add(e.inc)
	return e.price, nil
}
func (e *mockExchange) Balance(ctx context.Context, currency string) (decimal.Decimal, error) {
	return decimal.NewFromFloat(100.0), nil
}
func (e *mockExchange) Symbol(base, quote string) string {
	return fmt.Sprintf("%s%s", base, quote)
}
