package signal

import (
	"github.com/shopspring/decimal"
)

type Signal struct {
	Exchanges []string
	Base      string
	Quote     string
	Start     decimal.Decimal
	Targets   []decimal.Decimal
	Stop      decimal.Decimal
}

type Parser interface {
	Parse(text string) (*Signal, error)
}
