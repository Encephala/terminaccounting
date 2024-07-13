package ledgers

import (
	"database/sql/driver"
	"fmt"
)

func (lt *LedgerType) Scan(value interface{}) error {
	switch value {
	case int64(0):
		*lt = Income
	case int64(1):
		*lt = Expense
	case int64(2):
		*lt = Asset
	case int64(3):
		*lt = Liability
	case int64(4):
		*lt = Equity

	default:
		return fmt.Errorf("UNMARSHALLING INVALID LEDGER TYPE: %v", value)
	}

	return nil
}

func (lt LedgerType) Value() (driver.Value, error) {
	switch lt {
	case Income:
		return int64(0), nil
	case Expense:
		return int64(1), nil
	case Asset:
		return int64(2), nil
	case Liability:
		return int64(3), nil
	case Equity:
		return int64(4), nil
	}

	return nil, fmt.Errorf("MARSHALLING INVALID LEDGER TYPE: %v", lt)
}
