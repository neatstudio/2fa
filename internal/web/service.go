package web

import (
	"sort"
	"sync"
	"time"

	"github.com/gouki/tools/2fa/internal/model"
	"github.com/gouki/tools/2fa/internal/store"
	"github.com/gouki/tools/2fa/internal/totp"
)

type AccountView struct {
	Name      string `json:"name"`
	Group     string `json:"group"`
	Note      string `json:"note"`
	Code      string `json:"code"`
	Remaining int64  `json:"remaining"`
	UpdatedAt string `json:"updated_at"`
}

type AccountsResponse struct {
	Accounts []AccountView `json:"accounts"`
	Groups   []string      `json:"groups"`
}

type AccountInput struct {
	Name   string `json:"name"`
	Secret string `json:"secret"`
	Group  string `json:"group"`
	Note   string `json:"note"`
}

type AccountPatch struct {
	Secret *string `json:"secret"`
	Group  *string `json:"group"`
	Note   *string `json:"note"`
}

type Service struct {
	store *store.Store
	mu    sync.Mutex
}

func NewService(st *store.Store) *Service {
	return &Service{store: st}
}

func (s *Service) Snapshot(group string, now time.Time) (AccountsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.store.Load()
	if err != nil {
		return AccountsResponse{}, err
	}
	groups := accountGroups(accounts)
	if group != "" {
		accounts = accounts.FilterByGroup(group)
	}
	views, err := accountViews(accounts, now)
	if err != nil {
		return AccountsResponse{}, err
	}
	return AccountsResponse{Accounts: views, Groups: groups}, nil
}

func (s *Service) Add(input AccountInput, now time.Time) (AccountView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, err := model.NewAccount(input.Name, input.Secret, input.Group, input.Note, now)
	if err != nil {
		return AccountView{}, err
	}
	if _, err := totp.GenerateDefault(account.Secret, now); err != nil {
		return AccountView{}, err
	}
	accounts, err := s.store.Load()
	if err != nil {
		return AccountView{}, err
	}
	accounts, err = accounts.Add(account)
	if err != nil {
		return AccountView{}, err
	}
	if err := s.store.Save(accounts); err != nil {
		return AccountView{}, err
	}
	return accountView(account, now)
}

func (s *Service) Update(name string, patch AccountPatch, now time.Time) (AccountView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	changes := model.AccountChanges{Group: patch.Group, Note: patch.Note}
	if patch.Secret != nil && *patch.Secret != "" {
		normalized := model.NormalizeSecret(*patch.Secret)
		if _, err := totp.GenerateDefault(normalized, now); err != nil {
			return AccountView{}, err
		}
		changes.Secret = &normalized
	}
	accounts, err := s.store.Load()
	if err != nil {
		return AccountView{}, err
	}
	accounts, err = accounts.Update(name, changes, now)
	if err != nil {
		return AccountView{}, err
	}
	if err := s.store.Save(accounts); err != nil {
		return AccountView{}, err
	}
	account, _ := accounts.Find(name)
	return accountView(account, now)
}

func (s *Service) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.store.Load()
	if err != nil {
		return err
	}
	accounts, err = accounts.Delete(name)
	if err != nil {
		return err
	}
	return s.store.Save(accounts)
}

func accountViews(accounts model.Accounts, now time.Time) ([]AccountView, error) {
	views := make([]AccountView, 0, len(accounts))
	for _, account := range accounts {
		view, err := accountView(account, now)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func accountView(account model.Account, now time.Time) (AccountView, error) {
	code, err := totp.GenerateDefault(account.Secret, now)
	if err != nil {
		return AccountView{}, err
	}
	return AccountView{
		Name:      account.Name,
		Group:     account.Group,
		Note:      account.Note,
		Code:      code.Value,
		Remaining: code.Remaining,
		UpdatedAt: account.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func accountGroups(accounts model.Accounts) []string {
	seen := map[string]bool{}
	groups := make([]string, 0)
	for _, account := range accounts {
		if seen[account.Group] {
			continue
		}
		seen[account.Group] = true
		groups = append(groups, account.Group)
	}
	sort.Strings(groups)
	return groups
}
