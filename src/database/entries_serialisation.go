package database

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Don't ask me why we have to use this specific date as string format in Go
const DATE_FORMAT = "2006-01-02"

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
