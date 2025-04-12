package journals

import (
	"database/sql/driver"
	"fmt"
)

func (jt *JournalType) Scan(value any) error {
	switch value {
	case int64(0):
		*jt = INCOME
	case int64(1):
		*jt = EXPENSE
	case int64(2):
		*jt = CASHFLOW
	case int64(3):
		*jt = GENERAL

	default:
		return fmt.Errorf("UNMARSHALLING INVALID JOURNAL TYPE: %v", value)
	}

	return nil
}

func (jt JournalType) Value() (driver.Value, error) {
	switch jt {
	case INCOME:
		return int64(0), nil
	case EXPENSE:
		return int64(1), nil
	case CASHFLOW:
		return int64(2), nil
	case GENERAL:
		return int64(3), nil
	}

	return nil, fmt.Errorf("MARSHALLING INVALID JOURNAL TYPE: %v", jt)
}
