package signal

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

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

type parser struct {
	text *regexp.Regexp
	nums *regexp.Regexp
}

func NewParser() (Parser, error) {
	text, err := regexp.Compile(`[A-Z]+`)
	if err != nil {
		return nil, fmt.Errorf("signal: couldn't create regex: %w", err)
	}
	nums, err := regexp.Compile(`(?::|;)\s+([0-9]+(?:(?:.|,)[0-9]+)?)`)
	if err != nil {
		return nil, fmt.Errorf("signal: couldn't create regex: %w", err)
	}
	return &parser{
		text: text,
		nums: nums,
	}, nil
}

func (p *parser) Parse(text string) (*Signal, error) {
	if strings.Contains(text, "✅") || strings.Contains(text, "❌") {
		return nil, errors.New("trade has already started or finished")
	}
	text = strings.ToUpper(text)
	lines := strings.Split(text, "\n")
	if len(lines) < 5 {
		return nil, fmt.Errorf("trade hasn't enough lines: %d", len(lines))

	}
	exchanges := p.text.FindAllString(lines[0], -1)
	if len(exchanges) < 1 {
		return nil, fmt.Errorf("couldn't parse exchanges: %s", lines[0])
	}

	symbol := p.text.FindAllString(lines[1], -1)
	if len(symbol) < 2 {
		return nil, fmt.Errorf("couldn't parse symbol: %s", lines[1])
	}

	opts := &Signal{
		Exchanges: exchanges,
		Base:      strings.TrimSpace(symbol[0]),
		Quote:     strings.TrimSpace(symbol[1]),
	}

	rest := lines[2:]
	for i, line := range rest {
		matches := p.nums.FindStringSubmatch(line)
		if len(matches) < 2 {
			return nil, fmt.Errorf("price not found in line: %s", line)
		}
		match := strings.Replace(matches[1], ",", ".", 1)
		price, err := decimal.NewFromString(match)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse price %s: %w", match, err)
		}
		switch i {
		case 0:
			opts.Start = price
		case len(rest) - 1:
			opts.Stop = price
		default:
			opts.Targets = append(opts.Targets, price)
		}
	}
	return opts, nil
}
