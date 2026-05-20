package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gouki/tools/2fa/internal/model"
	"github.com/gouki/tools/2fa/internal/totp"
)

func RenderTable(accounts model.Accounts, now time.Time) (string, error) {
	headers := []string{"GROUP", "NAME", "CODE", "REMAINING", "NOTE"}
	rows := make([][]string, 0, len(accounts)+1)
	rows = append(rows, headers)

	for _, account := range accounts {
		code, err := totp.GenerateDefault(account.Secret, now)
		if err != nil {
			return "", fmt.Errorf("%s: %w", account.Name, err)
		}
		rows = append(rows, []string{
			account.Group,
			account.Name,
			code.Value,
			fmt.Sprintf("%d", code.Remaining),
			account.Note,
		})
	}

	if len(accounts) == 0 {
		rows = append(rows, []string{"-", "-", "-", "-", "no accounts"})
	}

	widths := make([]int, len(headers))
	for _, row := range rows {
		for i, cell := range row {
			if width := len(cell); width > widths[i] {
				widths[i] = width
			}
		}
	}

	var b strings.Builder
	writeRow := func(row []string) {
		for i, cell := range row {
			if i > 0 {
				b.WriteString("  ")
			}
			b.WriteString(padRight(cell, widths[i]))
		}
		b.WriteByte('\n')
	}

	writeRow(rows[0])
	separator := make([]string, len(headers))
	for i, width := range widths {
		separator[i] = strings.Repeat("-", width)
	}
	writeRow(separator)
	for _, row := range rows[1:] {
		writeRow(row)
	}
	return b.String(), nil
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}
