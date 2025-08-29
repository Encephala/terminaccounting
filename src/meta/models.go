package meta

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type SetupSchemaMsg struct{}

type Notes []string

func (n *Notes) Scan(value any) error {
	if value == nil {
		*n = make([]string, 0)
		return nil
	}

	converted, ok := value.(string)
	if !ok {
		return fmt.Errorf("UNMARSHALLING INVALID NOTES: %v", value)
	}

	return json.Unmarshal([]byte(converted), n)
}

func (n Notes) Value() (driver.Value, error) {
	binary, err := json.Marshal(n)
	result := string(binary)

	return result, err
}

func CompileNotes(input string) Notes {
	return strings.Split(input, "\n")
}

func (n Notes) Collapse() string {
	return strings.Join(n, "\n")
}
