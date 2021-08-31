package signal

import (
	"reflect"
	"testing"

	"github.com/shopspring/decimal"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		want    *Signal
		wantErr bool
	}{
		{
			name: "valid trade",
			msg: `🔥BINANCE
TFUEL-USDT
Entrada: 0.34141
Target 1: 0.36872 % 8
Target 2: 0.39262 % 15
Target 3: 0.42676 % 25
Target 4: 0.47797 % 40
Target 5: 0.54626 % 60
Stop Loss: 0.30044`,
			want: &Signal{
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
		{
			name: "multiple exchanges",
			msg: `🚨BINANCE / FTX
SOL / USDT
Entrada: 35.9 
Target 1: 38.8 % 8
Target 2: 41.6 % 16
Target 3: 44.9 % 25
Target 4: 50.3 % 40
Target 5: 57.4 % 60
Stop Loss: 31`,
			want: &Signal{
				Exchanges: []string{"BINANCE", "FTX"},
				Base:      "SOL",
				Quote:     "USDT",
				Start:     toDecimal("35.9"),
				Targets: []decimal.Decimal{
					toDecimal("38.8"),
					toDecimal("41.6"),
					toDecimal("44.9"),
					toDecimal("50.3"),
					toDecimal("57.4"),
				},
				Stop: toDecimal("31"),
			},
		},
		{
			name: "ignore checks",
			msg: `🔥BINANCE
TLM-USDT
Entrada: 0.2730
Target 1: 0.2948 % 8.✅
Target 2: 0.3140 % 15.✅
Target 3: 0.3413 % 25
Target 4: 0.3822 % 40
Target 5: 0.4368 % 60
Stop Loss: 0.2239`,
			wantErr: true,
		},
		{
			name: "ignore crosses",
			msg: `🔥BINANCE
LTC-USDT
Entrada: 206
Target 1: 222 % 8
Target 2: 237 % 15
Target 3: 258 % 25
Target 4: 288 % 40
Target 5: 330 % 60
Stop Loss: 185 %.❌`,
			wantErr: true,
		},
		{
			name: "not supported exchanges",
			msg: `🔥gate.io
ZPT-USDT
Entrada: 0.0009783
Target 1: 0.0010566 % 8
Target 2: 0.0011250 % 15
Target 3: 0.0012229 % 25
Target 4: 0.0013696 % 40
Target 5: 0.0015653 % 60
Stop Loss: 0.0008609`,
			wantErr: true,
		},
	}

	parser, err := NewParser()
	if err != nil {
		t.Fatal(err)
	}

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
