package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/zhanhd/gitra/internal/domain"
)

// newRandomID produces prefixed, collision-resistant identifiers such as
// bnd_9f2c... . IDs are opaque to users; only the prefix is contractual.
func newRandomID(prefix string) (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return prefix + hex.EncodeToString(buf[:]), nil
}

func generateBindingID() (domain.BindingID, error) {
	id, err := newRandomID("bnd_")
	if err != nil {
		return "", err
	}
	return domain.BindingID(id), nil
}

func generateAccountID() (domain.AccountID, error) {
	id, err := newRandomID("acc_")
	if err != nil {
		return "", err
	}
	return domain.AccountID(id), nil
}
