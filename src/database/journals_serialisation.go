package database

import (
	"database/sql/driver"
	"fmt"
)

func (jt *JournalType) Scan(value any) error {
	switch value {
	case int64(0):
		*jt = INCOMEJOURNAL
	case int64(1):
		*jt = EXPENSEJOURNAL
	case int64(2):
		*jt = CASHFLOWJOURNAL
	case int64(3):
		*jt = GENERALJOURNAL

	default:
		return fmt.Errorf("UNMARSHALLING INVALID JOURNAL TYPE: %v", value)
	}

	return nil
}

func (jt JournalType) Value() (driver.Value, error) {
	switch jt {
	case INCOMEJOURNAL:
		return int64(0), nil
	case EXPENSEJOURNAL:
		return int64(1), nil
	case CASHFLOWJOURNAL:
		return int64(2), nil
	case GENERALJOURNAL:
		return int64(3), nil
	}

	return nil, fmt.Errorf("MARSHALLING INVALID JOURNAL TYPE: %v", jt)
}
