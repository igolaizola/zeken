package parser

import (
	"errors"

	"github.com/igolaizola/zeken/pkg/signal"
	"github.com/igolaizola/zeken/pkg/signal/parser/cryptosignals"
	"github.com/igolaizola/zeken/pkg/signal/parser/json"
)

var ErrNotFound = errors.New("parser: not found")

func NewParser(name string) (signal.Parser, error) {
	switch name {
	case "json":
		return json.Parser{}, nil
	case "cryptosignals":
		return cryptosignals.NewParser()
	default:
		return nil, ErrNotFound
	}
}
