package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/jsanca/go-folio/internal/domain"
)

func encodeCursor(c domain.SyncCursor) string {
	b, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(b)
}

func decodeCursor(raw string) (domain.SyncCursor, error) {
	b, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		return domain.SyncCursor{}, fmt.Errorf("%w: %v", ErrInvalidCursor, err)
	}
	var c domain.SyncCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return domain.SyncCursor{}, fmt.Errorf("%w: %v", ErrInvalidCursor, err)
	}
	if c.UpdatedAt.IsZero() {
		return domain.SyncCursor{}, fmt.Errorf("%w: missing updatedAt", ErrInvalidCursor)
	}
	return c, nil
}
