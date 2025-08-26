package database

import (
	"database/sql/driver"
	"fmt"
)

func (lt *LedgerType) Scan(value any) error {
	switch value {
	case int64(0):
		*lt = INCOMELEDGER
	case int64(1):
		*lt = EXPENSELEDGER
	case int64(2):
		*lt = ASSETLEDGER
	case int64(3):
		*lt = LIABILITYLEDGER
	case int64(4):
		*lt = EQUITYLEDGER

	default:
		return fmt.Errorf("UNMARSHALLING INVALID LEDGER TYPE: %v", value)
	}

	return nil
}

func (lt LedgerType) Value() (driver.Value, error) {
	switch lt {
	case INCOMELEDGER:
		return int64(0), nil
	case EXPENSELEDGER:
		return int64(1), nil
	case ASSETLEDGER:
		return int64(2), nil
	case LIABILITYLEDGER:
		return int64(3), nil
	case EQUITYLEDGER:
		return int64(4), nil
	}

	return nil, fmt.Errorf("MARSHALLING INVALID LEDGER TYPE: %v", lt)
}
