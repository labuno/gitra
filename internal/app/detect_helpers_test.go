package app

import (
	"context"
	"os"

	"github.com/zhanhd/gitra/internal/ports"
)

type stubTokensOK struct{ token ports.LoginToken }

func (s stubTokensOK) Acquire(context.Context, ports.TokenRequest) (ports.LoginToken, error) {
	return s.token, nil
}

func osMkdirAll(dir string) error { return os.MkdirAll(dir, 0o700) }

func osWriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
