package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/adshao/go-binance/v2"
	"github.com/igolaizola/zeken/pkg/exchange"
	"github.com/shopspring/decimal"
)

type binanceExchange struct {
	client *binance.Client
	log    func(v ...interface{})
	debug  bool
}

const (
	decimalPrecision = 8
)

var zero = decimal.Decimal{}

func New(log func(v ...interface{}), apiKey, apiSecret string, debug bool) exchange.Exchange {
	cli := binance.NewClient(apiKey, apiSecret)
	cli.NewSetServerTimeService().Do(context.Background())
	return &binanceExchange{
		client: cli,
		log:    log,
		debug:  debug,
	}
}

func (c *binanceExchange) Symbol(base, quote string) string {
	return fmt.Sprintf("%s%s", base, quote)
}

func (c *binanceExchange) Buy(ctx context.Context, symbol string, quoteQuantity, price decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	currentPrice, err := c.Price(ctx, symbol)
	if err != nil {
		return zero, zero, err
	}
	// TODO(igolaizola): make this value configurable
	if currentPrice.LessThan(price.Mul(decimal.NewFromFloat(0.95))) {
		return zero, zero, fmt.Errorf("binance: current price is lower than minimum price: %s %s", currentPrice, price)
	}
	quoteQty, qty, err := c.buyLimit(ctx, symbol, quoteQuantity, currentPrice)
	if err != nil {
		c.log(fmt.Errorf("binance: buy limit failed, falling back to buy market: %w", err))
		return c.buyMarket(ctx, symbol, quoteQuantity)
	}
	return quoteQty, qty, nil
}

func (e *binanceExchange) Sell(ctx context.Context, symbol string, quantity decimal.Decimal) (decimal.Decimal, error) {
	quantity = quantity.Round(decimalPrecision)
	order, err := e.client.NewCreateOrderService().Symbol(symbol).
		Side(binance.SideTypeSell).
		Type(binance.OrderTypeMarket).
		Quantity(quantity.String()).
		Do(context.Background())
	if err != nil {
		return zero, err
	}
	// Debug
	if e.debug {
		js, _ := json.Marshal(order)
		e.log("sell_order", string(js))
	}
	id := order.OrderID
	for {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}
		ok, quoteQty, _, err := e.getOrder(ctx, symbol, strconv.Itoa(int(id)))
		var netErr net.Error
		if errors.As(err, &netErr) {
			continue
		}
		if err != nil {
			return zero, fmt.Errorf("binance: couldn't get sell order: %w", err)
		}
		if ok {
			return quoteQty, nil
		}
	}
}

func (e *binanceExchange) buyMarket(ctx context.Context, symbol string, quoteQuantity decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	quoteQty := quoteQuantity.Round(decimalPrecision)
	order, err := e.client.NewCreateOrderService().Symbol(symbol).
		Side(binance.SideTypeBuy).
		Type(binance.OrderTypeMarket).
		QuoteOrderQty(quoteQty.String()).
		Do(context.Background())
	if err != nil {
		return zero, zero, err
	}
	// Debug
	if e.debug {
		js, _ := json.Marshal(order)
		e.log("buy_market_order:", string(js))
	}
	id := order.OrderID
	for {
		select {
		case <-ctx.Done():
			return zero, zero, ctx.Err()
		default:
		}
		ok, quoteQty, qty, err := e.getOrder(ctx, symbol, strconv.Itoa(int(id)))
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			continue
		}
		if err != nil {
			return zero, zero, fmt.Errorf("binance: couldn't get buy market order: %w", err)
		}
		if ok {
			return quoteQty, qty, nil
		}
	}
}

func (e *binanceExchange) buyLimit(ctx context.Context, symbol string, quoteQuantity, price decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	precision := decimalPrecision
	info, err := e.client.NewExchangeInfoService().Symbol(symbol).Do(ctx)
	if err != nil {
		return zero, zero, fmt.Errorf("binance: couldn't get exchange info for %s: %w", symbol, err)
	}
	for _, s := range info.Symbols {
		if s.Symbol != symbol {
			continue
		}
		lotSize := s.LotSizeFilter()
		split := strings.Split(lotSize.StepSize, ".")
		if len(split) != 2 {
			return zero, zero, errors.New("binance: couldn't parse step size %s")
		}
		precision = len(strings.TrimRight(split[1], "0"))
	}

	qty := quoteQuantity.Div(price).Round(int32(precision))
	order, err := e.client.NewCreateOrderService().Symbol(symbol).
		Side(binance.SideTypeBuy).
		Type(binance.OrderTypeLimit).
		TimeInForce(binance.TimeInForceTypeFOK).
		Quantity(qty.String()).
		Price(price.String()).
		Do(context.Background())
	if err != nil {
		return zero, zero, fmt.Errorf("binance: couldn't get buy limit order (%s %s): %w", qty, price, err)
	}
	// Debug
	if e.debug {
		js, _ := json.Marshal(order)
		e.log("buy_limit_order:", string(js))
	}
	id := order.OrderID
	for {
		select {
		case <-ctx.Done():
			return zero, zero, ctx.Err()
		default:
		}
		ok, quoteQty, qty, err := e.getOrder(ctx, symbol, strconv.Itoa(int(id)))
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			continue
		}
		if err != nil {
			return zero, zero, fmt.Errorf("binance: couldn't get buy limit order: %w", err)
		}
		if ok {
			return quoteQty, qty, nil
		}
	}
}

func (e *binanceExchange) CreateStopLimit(ctx context.Context, symbol string, quantity, target, stop decimal.Decimal) (string, []string, error) {
	precision := decimalPrecision
	info, err := e.client.NewExchangeInfoService().Symbol(symbol).Do(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("binance: couldn't get exchange info for %s: %w", symbol, err)
	}
	for _, s := range info.Symbols {
		if s.Symbol != symbol {
			continue
		}
		priceFilter := s.PriceFilter()
		split := strings.Split(priceFilter.TickSize, ".")
		if len(split) != 2 {
			return "", nil, errors.New("binance: couldn't parse step size %s")
		}
		precision = len(strings.TrimRight(split[1], "0"))
	}
	target = target.Round(int32(precision))
	stop = stop.Round(int32(precision))
	limit := stop.Mul(decimal.NewFromFloat(0.99)).Round(int32(precision))

	order, err := e.client.NewCreateOCOService().Symbol(symbol).
		Side(binance.SideTypeSell).
		StopLimitTimeInForce(binance.TimeInForceTypeGTC).
		Quantity(quantity.String()).
		Price(target.String()).
		StopPrice(stop.String()).
		StopLimitPrice(limit.String()).
		Do(context.Background())
	if err != nil {
		return "", nil, err
	}
	if e.debug {
		js, _ := json.Marshal(order)
		e.log("stop_limit_order:", string(js))
	}
	var orderIDs []string
	for _, o := range order.Orders {
		orderIDs = append(orderIDs, strconv.Itoa(int(o.OrderID)))
	}
	orderListID := strconv.Itoa(int(order.OrderListID))
	return orderListID, orderIDs, nil
}

func (e *binanceExchange) CancelStopLimit(ctx context.Context, symbol string, id string) error {
	orderListID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("binance: invalid id %s: %w", id, err)
	}
	_, err = e.client.NewCancelOCOService().Symbol(symbol).
		OrderListID(int64(orderListID)).
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (e *binanceExchange) Status(ctx context.Context, symbol string, id string) (bool, decimal.Decimal, error) {
	ok, quoteQty, _, err := e.getOrder(ctx, symbol, id)
	return ok, quoteQty, err
}

func (e *binanceExchange) getOrder(ctx context.Context, symbol string, id string) (bool, decimal.Decimal, decimal.Decimal, error) {
	orderID, err := strconv.Atoi(id)
	if err != nil {
		return false, zero, zero, fmt.Errorf("binance: invalid id %s: %w", id, err)
	}
	order, err := e.client.NewGetOrderService().Symbol(symbol).
		OrderID(int64(orderID)).Do(ctx)
	if err != nil {
		return false, zero, zero, fmt.Errorf("binance: couldn't get order: %w", err)
	}
	switch order.Status {
	// The order has been accepted by the engine.
	case binance.OrderStatusTypeNew:
		return false, zero, zero, nil
	// A part of the order has been filled.
	case binance.OrderStatusTypePartiallyFilled:
		return false, zero, zero, nil
	// The order has been completed.
	case binance.OrderStatusTypeFilled:
		if e.debug {
			js, _ := json.Marshal(order)
			e.log("order_filled", string(js))
		}
		qty, err := decimal.NewFromString(order.ExecutedQuantity)
		if err != nil {
			return false, zero, zero, fmt.Errorf("binance: couldn't parse quantity: %s: %w", order.ExecutedQuantity, err)
		}
		quoteQty, err := decimal.NewFromString(order.CummulativeQuoteQuantity)
		if err != nil {
			return false, zero, zero, fmt.Errorf("binance: couldn't parse price: %s: %w", order.CummulativeQuoteQuantity, err)
		}
		return true, quoteQty, qty, nil
	case binance.OrderStatusTypeCanceled:
		return false, zero, zero, fmt.Errorf("binance: %w", exchange.ErrOrderCanceled)
	default:
	}
	return false, zero, zero, fmt.Errorf("status %s", order.Status)
}

func (e *binanceExchange) Price(ctx context.Context, symbol string) (decimal.Decimal, error) {
	prices, err := e.client.NewListPricesService().Symbol(symbol).Do(ctx)
	if err != nil {
		return zero, fmt.Errorf("binance: couldn't get price for %s: %w", symbol, err)
	}
	for _, p := range prices {
		if p.Symbol != symbol {
			continue
		}
		price, err := decimal.NewFromString(p.Price)
		if err != nil {
			return zero, fmt.Errorf("binance: couldn't parse price: %s: %w", p.Price, err)
		}
		return price, nil
	}
	return zero, fmt.Errorf("binance: price for %s not found: %w", symbol, err)
}

func (e *binanceExchange) Balance(ctx context.Context, currency string) (decimal.Decimal, error) {
	acc, err := e.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return zero, fmt.Errorf("binance: couldn't get account: %w", err)
	}
	for _, b := range acc.Balances {
		if b.Asset == currency {
			balance, err := decimal.NewFromString(b.Free)
			if err != nil {
				return zero, fmt.Errorf("binance: couldn't parse balance %s: %w", b.Free, err)
			}
			return balance, nil
		}
	}
	return zero, fmt.Errorf("binance: balance for %s not found", currency)
}
