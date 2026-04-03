package httpserver

import (
	"fmt"
	"time"
)

func NotEmpty(fieldName string, value string) error {
	if value == "" {
		return fmt.Errorf("invalid \"%s\" field", fieldName)
	}
	return nil
}

func NotNegative(fieldName string, val int64) error {
	if val < 0 {
		return fmt.Errorf("invalid \"%s\" parameter", fieldName)
	}
	return nil
}

func NotNegativeField(fieldName string, val int64) error {
	if val < 0 {
		return fmt.Errorf("invalid \"%s\" field", fieldName)
	}
	return nil
}

func ParseRFC3339(fieldName string, value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid \"%s\" field", fieldName)
	}
	return t, nil
}
