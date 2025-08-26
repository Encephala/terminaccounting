package database

import (
	"database/sql/driver"
	"fmt"
)

func (at *AccountType) Scan(value any) error {
	switch value {
	case int64(0):
		*at = DEBTOR
	case int64(1):
		*at = CREDITOR

	default:
		return fmt.Errorf("UNMARSHALLING INVALID ACCOUNT TYPE: %v", value)
	}

	return nil
}

func (at AccountType) Value() (driver.Value, error) {
	switch at {
	case DEBTOR:
		return int64(0), nil
	case CREDITOR:
		return int64(1), nil
	}

	return nil, fmt.Errorf("MARSHALLING INVALID ACCOUNT TYPE: %v", at)
}
