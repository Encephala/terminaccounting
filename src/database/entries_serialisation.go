package database

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
)

func (er *DecimalValue) Scan(value any) error {
	switch value := value.(type) {
	case []byte:
		if len(value) != 16 {
			return fmt.Errorf("UNMARSHALLING `DecimalValue` BUT DID NOT GET 16 BYTES, GOT: %+v", value)
		}

		er.Whole = int64(binary.LittleEndian.Uint64(value))
		er.Fractional = binary.LittleEndian.Uint64(value[8:])

	default:
		return fmt.Errorf("UNMARSHALLING INVALID `DecimalValue`: %+v", value)
	}

	return nil
}

func (er DecimalValue) Value() (driver.Value, error) {
	result := make([]byte, 16)

	binary.LittleEndian.PutUint64(result, uint64(er.Whole))
	binary.LittleEndian.PutUint64(result[8:], er.Fractional)

	return result, nil
}
