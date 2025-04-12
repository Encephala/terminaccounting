package ledgers

import (
	"database/sql/driver"
	"fmt"
)

func (lt *LedgerType) Scan(value any) error {
	switch value {
	case int64(0):
		*lt = INCOME
	case int64(1):
		*lt = EXPENSE
	case int64(2):
		*lt = ASSET
	case int64(3):
		*lt = LIABILITY
	case int64(4):
		*lt = EQUITY

	default:
		return fmt.Errorf("UNMARSHALLING INVALID LEDGER TYPE: %v", value)
	}

	return nil
}

func (lt LedgerType) Value() (driver.Value, error) {
	switch lt {
	case INCOME:
		return int64(0), nil
	case EXPENSE:
		return int64(1), nil
	case ASSET:
		return int64(2), nil
	case LIABILITY:
		return int64(3), nil
	case EQUITY:
		return int64(4), nil
	}

	return nil, fmt.Errorf("MARSHALLING INVALID LEDGER TYPE: %v", lt)
}
