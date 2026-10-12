package gitcli

import (
	"context"
	"fmt"
	"os"

	"github.com/zhanhd/gitra/internal/domain"
)

func (a *Adapter) InitializeRepository(ctx context.Context, path string) error {
	dir, err := expandUser(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s is not a folder", domain.ErrInvalid, dir)
	}
	result, err := a.run(ctx, dir, "init", "--initial-branch=main")
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("git init: %s", firstLine(result.Stderr))
	}
	return nil
}
