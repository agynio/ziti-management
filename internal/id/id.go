package id

import "github.com/google/uuid"

func ShortUUID() string {
	value := uuid.NewString()
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}
