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
	var result Notes

	lines := strings.Split(input, "\n")

	i := 0
	for !allNextLinesBlank(lines[i:]) {
		result = append(result, lines[i])
		i++
	}

	return result
}

func allNextLinesBlank(nextLines []string) bool {
	for _, line := range nextLines {
		if line != "" {
			return false
		}
	}

	return true
}

// Joins the Notes together into a newline-delimited string.
func (n Notes) Collapse() string {
	return strings.Join(n, "\n")
}
