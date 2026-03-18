package store

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	defaultPageSize int32 = 50
	maxPageSize     int32 = 100
)

func normalizePageSize(size int32) int32 {
	if size <= 0 {
		return defaultPageSize
	}
	if size > maxPageSize {
		return maxPageSize
	}
	return size
}

type pageToken struct {
	ZitiIdentityID string `json:"ziti_identity_id"`
}

func EncodePageToken(zitiIdentityID string) (string, error) {
	payload := pageToken{ZitiIdentityID: zitiIdentityID}
	buf, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func DecodePageToken(token string) (string, error) {
	if token == "" {
		return "", errors.New("empty token")
	}
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	var payload pageToken
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("unmarshal token: %w", err)
	}
	if payload.ZitiIdentityID == "" {
		return "", errors.New("token missing ziti identity id")
	}
	return payload.ZitiIdentityID, nil
}
