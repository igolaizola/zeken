package trade

import "time"

type Store interface {
	List(from time.Time, to time.Time, finished bool) ([]*Trade, error)
	Update(*Trade) error
	Delete(*Trade) error
}
