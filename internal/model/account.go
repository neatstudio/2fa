package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const DefaultGroup = "default"

var (
	ErrDuplicateName = errors.New("account name already exists")
	ErrNotFound      = errors.New("account not found")
)

type Account struct {
	Name      string    `json:"name"`
	Group     string    `json:"group"`
	Note      string    `json:"note,omitempty"`
	Secret    string    `json:"secret"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AccountChanges struct {
	Group  *string
	Note   *string
	Secret *string
}

type Accounts []Account

func NewAccount(name, secret, group, note string, now time.Time) (Account, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Account{}, errors.New("name is required")
	}

	secret = NormalizeSecret(secret)
	if secret == "" {
		return Account{}, errors.New("secret is required")
	}

	group = normalizeGroup(group)
	now = now.UTC()
	return Account{
		Name:      name,
		Group:     group,
		Note:      strings.TrimSpace(note),
		Secret:    secret,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func NormalizeSecret(secret string) string {
	var b strings.Builder
	b.Grow(len(secret))
	for _, r := range secret {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(b.String())
}

func (accounts Accounts) Add(account Account) (Accounts, error) {
	if account.Name == "" {
		return nil, errors.New("name is required")
	}
	if _, ok := accounts.Find(account.Name); ok {
		return nil, fmt.Errorf("%w: %s", ErrDuplicateName, account.Name)
	}
	next := make(Accounts, 0, len(accounts)+1)
	next = append(next, accounts...)
	next = append(next, account)
	return next, nil
}

func (accounts Accounts) Update(name string, changes AccountChanges, now time.Time) (Accounts, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	next := make(Accounts, len(accounts))
	copy(next, accounts)

	for i := range next {
		if next[i].Name != name {
			continue
		}
		if changes.Group != nil {
			next[i].Group = normalizeGroup(*changes.Group)
		}
		if changes.Note != nil {
			next[i].Note = strings.TrimSpace(*changes.Note)
		}
		if changes.Secret != nil {
			secret := NormalizeSecret(*changes.Secret)
			if secret == "" {
				return nil, errors.New("secret is required")
			}
			next[i].Secret = secret
		}
		next[i].UpdatedAt = now.UTC()
		return next, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
}

func (accounts Accounts) Delete(name string) (Accounts, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	next := make(Accounts, 0, len(accounts))
	deleted := false
	for _, account := range accounts {
		if account.Name == name {
			deleted = true
			continue
		}
		next = append(next, account)
	}
	if !deleted {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return next, nil
}

func (accounts Accounts) Find(name string) (Account, bool) {
	for _, account := range accounts {
		if account.Name == name {
			return account, true
		}
	}
	return Account{}, false
}

func (accounts Accounts) FilterByGroup(group string) Accounts {
	group = strings.TrimSpace(group)
	filtered := make(Accounts, 0, len(accounts))
	for _, account := range accounts {
		if account.Group == group {
			filtered = append(filtered, account)
		}
	}
	return filtered
}

func (accounts Accounts) ValidateUniqueNames() error {
	seen := make(map[string]struct{}, len(accounts))
	for _, account := range accounts {
		if account.Name == "" {
			return errors.New("account name is required")
		}
		if _, ok := seen[account.Name]; ok {
			return fmt.Errorf("%w: %s", ErrDuplicateName, account.Name)
		}
		seen[account.Name] = struct{}{}
	}
	return nil
}

func normalizeGroup(group string) string {
	group = strings.TrimSpace(group)
	if group == "" {
		return DefaultGroup
	}
	return group
}
