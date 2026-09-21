package database

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Don't ask me why we have to use this specific date as string format in Go
const DATE_FORMAT = "06-01-02"

func (et *EntryType) Scan(value any) error {
	switch value {
	case int64(0):
		*et = INCOMEENTRY
	case int64(1):
		*et = EXPENSEENTRY
	case int64(2):
		*et = CASHFLOWENTRY
	case int64(3):
		*et = GENERALENTRY

	default:
		return fmt.Errorf("UNMARSHALLING INVALID ENTRY TYPE: %v", value)
	}

	return nil
}

func (et EntryType) Value() (driver.Value, error) {
	switch et {
	case INCOMEENTRY:
		return int64(0), nil
	case EXPENSEENTRY:
		return int64(1), nil
	case CASHFLOWENTRY:
		return int64(2), nil
	case GENERALENTRY:
		return int64(3), nil
	}

	return nil, fmt.Errorf("MARSHALLING INVALID ENTRY TYPE: %v", et)
}

func (d *Date) Scan(value any) error {
	switch value := value.(type) {
	case string:
		parsed, err := time.Parse(DATE_FORMAT, value)
		if err != nil {
			return err
		}

		*d = Date(parsed)

	default:
		return fmt.Errorf("UNMARSHALLING INVALID DATE: %#v", value)
	}

	return nil
}

func (d Date) Value() (driver.Value, error) {
	return time.Time(d).Format(DATE_FORMAT), nil
}
