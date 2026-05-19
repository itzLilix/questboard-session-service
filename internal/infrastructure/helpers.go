package infrastructure

import "strings"

func nullString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}