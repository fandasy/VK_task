package e

import "fmt"

func Wrap(msg string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", msg, err)
}

func String(msg string, err error) string {
	if err == nil {
		return msg
	}

	return fmt.Sprintf("%s: %v", msg, err)
}
