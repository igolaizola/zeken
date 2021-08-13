package inmem

import (
	"sync"
	"time"

	"github.com/igolaizola/zeken/pkg/trade"
)

type Store struct {
	trades sync.Map
}

func (s *Store) List(from time.Time, to time.Time, finished bool) ([]trade.Trade, error) {
	var trades []trade.Trade
	s.trades.Range(func(key interface{}, value interface{}) bool {
		t := value.(trade.Trade)
		if t.StartTime.Before(from) {
			return true
		}
		if t.StartTime.After(to) {
			return true
		}
		hasFinished := t.EndTime != time.Time{}
		if hasFinished != finished {
			return true
		}
		trades = append(trades, t)
		return true
	})
	return trades, nil
}

func (s *Store) Save(t trade.Trade) error {
	s.trades.Store(t.StartTime, t)
	return nil
}

func (s *Store) Delete(t trade.Trade) error {
	s.trades.Delete(t.StartTime)
	return nil
}
