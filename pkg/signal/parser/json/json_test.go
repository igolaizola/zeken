package json

import (
	"reflect"
	"testing"

	"github.com/igolaizola/zeken/pkg/signal"
	"github.com/shopspring/decimal"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		want    *signal.Signal
		wantErr bool
	}{
		{
			name: "valid trade",
			msg: `{
	"exchanges": ["BINANCE"],
	"base": "TFUEL",
	"quote": "USDT",
	"start": "0.34141",
	"targets": [
		"0.36872",
		"0.39262",
		"0.42676",
		"0.47797",
		"0.54626"
	],
	"stop": "0.30044"
}`,
			want: &signal.Signal{
				Exchanges: []string{"BINANCE"},
				Base:      "TFUEL",
				Quote:     "USDT",
				Start:     toDecimal("0.34141"),
				Targets: []decimal.Decimal{
					toDecimal("0.36872"),
					toDecimal("0.39262"),
					toDecimal("0.42676"),
					toDecimal("0.47797"),
					toDecimal("0.54626"),
				},
				Stop: toDecimal("0.30044"),
			},
		},
	}

	parser := Parser{}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parser.Parse(tt.msg)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*opts, *tt.want) {
				t.Errorf("got: %v, want: %v", opts, tt.want)
			}
		})
	}
}

func toDecimal(value string) decimal.Decimal {
	d, err := decimal.NewFromString(value)
	if err != nil {
		panic(err)
	}
	return d
}
