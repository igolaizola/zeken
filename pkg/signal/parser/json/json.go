package json

import (
	"encoding/json"
	"fmt"

	"github.com/igolaizola/zeken/pkg/signal"
	"github.com/shopspring/decimal"
)

type Parser struct{}

type jsonSignal struct {
	Exchanges []string `json:"exchanges"`
	Base      string   `json:"base"`
	Quote     string   `json:"quote"`
	Start     string   `json:"start"`
	Targets   []string `json:"targets"`
	Stop      string   `json:"stop"`
}

func (p Parser) Parse(text string) (*signal.Signal, error) {
	var js jsonSignal
	if err := json.Unmarshal([]byte(text), &js); err != nil {
		return nil, fmt.Errorf("json: couldn't parse signal (%s): %w", text, err)
	}
	s := &signal.Signal{
		Exchanges: js.Exchanges,
		Base:      js.Base,
		Quote:     js.Quote,
		Targets:   make([]decimal.Decimal, len(js.Targets)),
	}
	var err error
	s.Start, err = decimal.NewFromString(js.Start)
	if err != nil {
		return nil, fmt.Errorf("json: couldn't parse start price (%s): %w", s.Start, err)
	}
	s.Stop, err = decimal.NewFromString(js.Stop)
	if err != nil {
		return nil, fmt.Errorf("json: couldn't parse stop price (%s): %w", s.Stop, err)
	}
	for i, target := range js.Targets {
		s.Targets[i], err = decimal.NewFromString(target)
		if err != nil {
			return nil, fmt.Errorf("json: couldn't parse target %d price (%s): %w", i+1, target, err)
		}
	}
	return s, nil
}
