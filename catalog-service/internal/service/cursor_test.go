package service

import (
	"errors"
	"testing"
	"time"

	"github.com/jsanca/go-folio/internal/domain"
)

func TestCursor_EncodeAndDecode_RoundTrip(t *testing.T) {
	original := domain.SyncCursor{
		UpdatedAt: time.Date(2026, 5, 14, 20, 5, 0, 0, time.UTC),
		ID:        1234,
	}
	encoded := encodeCursor(original)
	if encoded == "" {
		t.Fatal("encoded cursor must not be empty")
	}

	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if !decoded.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("updatedAt mismatch: got %v, want %v", decoded.UpdatedAt, original.UpdatedAt)
	}
	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %d, want %d", decoded.ID, original.ID)
	}
}

func TestCursor_DecodeInvalidBase64_ReturnsErrInvalidCursor(t *testing.T) {
	_, err := decodeCursor("not-valid-base64!!!")
	if !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestCursor_DecodeValidBase64ButInvalidJSON_ReturnsErrInvalidCursor(t *testing.T) {
	encoded := "dGhpcyBpcyBub3QganNvbg==" // base64("this is not json")
	_, err := decodeCursor(encoded)
	if !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestCursor_DecodeMissingUpdatedAt_ReturnsErrInvalidCursor(t *testing.T) {
	encoded := "eyJpZCI6MTIzNH0=" // base64({"id":1234}) — missing updatedAt
	_, err := decodeCursor(encoded)
	if !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("expected ErrInvalidCursor for missing updatedAt, got %v", err)
	}
}

func TestCursor_IsDifferentForDifferentPositions(t *testing.T) {
	c1 := encodeCursor(domain.SyncCursor{UpdatedAt: time.Now(), ID: 1})
	c2 := encodeCursor(domain.SyncCursor{UpdatedAt: time.Now(), ID: 2})
	if c1 == c2 {
		t.Error("cursors for different IDs must not be equal")
	}
}
