package ui

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gouki/tools/2fa/internal/model"
)

func Watch(ctx context.Context, out io.Writer, accounts model.Accounts) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		if err := renderScreen(out, accounts, time.Now()); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func renderScreen(out io.Writer, accounts model.Accounts, now time.Time) error {
	table, err := RenderTable(accounts, now)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "\033[H\033[2J%s", table)
	return err
}
