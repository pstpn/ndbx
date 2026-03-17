package httpserver

import (
	"errors"
	"fmt"
	"time"
)

func NotEmpty(fieldName string, value string) error {
	if value == "" {
		return fmt.Errorf("invalid \"%s\" field", fieldName)
	}
	return nil
}

func NotNegative(limit int64, offset int64) error {
	if limit < 0 {
		return errors.New("invalid \"limit\" parameter")
	}
	if offset < 0 {
		return errors.New("invalid \"offset\" parameter")
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
