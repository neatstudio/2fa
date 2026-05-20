package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gouki/tools/2fa/internal/model"
)

const (
	dirPerm  os.FileMode = 0o700
	filePerm os.FileMode = 0o600
)

type Store struct {
	path string
}

type diskData struct {
	Accounts model.Accounts `json:"accounts"`
}

func New(path string) *Store {
	return &Store{path: path}
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".2fa", "accounts.json"), nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (model.Accounts, error) {
	if s.path == "" {
		return nil, errors.New("store path is required")
	}
	if err := prepareStoreDir(s.path); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return model.Accounts{}, nil
	}
	if err != nil {
		return nil, err
	}
	if err := chmodIfSupported(s.path, filePerm); err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return model.Accounts{}, nil
	}

	var stored diskData
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, fmt.Errorf("read accounts: %w", err)
	}
	if stored.Accounts == nil {
		stored.Accounts = model.Accounts{}
	}
	if err := stored.Accounts.ValidateUniqueNames(); err != nil {
		return nil, err
	}
	return stored.Accounts, nil
}

func (s *Store) Save(accounts model.Accounts) error {
	if s.path == "" {
		return errors.New("store path is required")
	}
	if err := accounts.ValidateUniqueNames(); err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := prepareStoreDir(s.path); err != nil {
		return err
	}

	data, err := json.MarshalIndent(diskData{Accounts: accounts}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmpFile, err := os.CreateTemp(dir, ".accounts.json.tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if err := chmodIfSupported(tmpName, filePerm); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return err
	}
	cleanup = false
	if err := chmodIfSupported(s.path, filePerm); err != nil {
		return err
	}

	if dirFile, err := os.Open(dir); err == nil {
		_ = dirFile.Sync()
		_ = dirFile.Close()
	}
	return nil
}

func ensurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	return chmodIfSupported(dir, dirPerm)
}

func prepareStoreDir(path string) error {
	dir := filepath.Dir(path)
	if isDefaultPath(path) {
		return ensurePrivateDir(dir)
	}

	if _, err := os.Stat(dir); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return ensurePrivateDir(dir)
}

func isDefaultPath(path string) bool {
	defaultPath, err := DefaultPath()
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path == defaultPath
	}
	absDefault, err := filepath.Abs(defaultPath)
	if err != nil {
		return path == defaultPath
	}
	return absPath == absDefault
}

func chmodIfSupported(path string, perm os.FileMode) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(path, perm)
}
