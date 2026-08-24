package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Spondon16/mailme/api"
)

var (
	configDir    string
	accountsFile string
)

func init() {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	configDir = filepath.Join(dir, "mailme")
	accountsFile = filepath.Join(configDir, "accounts.json")
}

func ensureDir() error {
	return os.MkdirAll(configDir, 0700)
}

// AccountRecord is the JSON-serializable account stored on disk.
type AccountRecord struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Token     string `json:"token,omitempty"`
	Provider  string `json:"provider"`
	CreatedAt string `json:"created_at,omitempty"`
}

// LoadAccounts reads all saved accounts from disk.
func LoadAccounts() ([]AccountRecord, error) {
	if err := ensureDir(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(accountsFile)
	if os.IsNotExist(err) || len(data) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read accounts: %w", err)
	}

	var accounts []AccountRecord
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, fmt.Errorf("parse accounts: %w", err)
	}
	return accounts, nil
}

func saveAccounts(accounts []AccountRecord) error {
	if err := ensureDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal accounts: %w", err)
	}

	// Atomic write: write to unique temp file then rename
	tmp := filepath.Join(configDir, fmt.Sprintf("accounts.%d.tmp", time.Now().UnixNano()))
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write temp accounts: %w", err)
	}

	if err := os.Rename(tmp, accountsFile); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename accounts: %w", err)
	}
	return nil
}

// AddAccount saves an account to disk.
func AddAccount(acc *api.Account) error {
	accounts, err := LoadAccounts()
	if err != nil {
		return err
	}

	// Remove existing with same email
	filtered := make([]AccountRecord, 0)
	for _, a := range accounts {
		if a.Email != acc.Email {
			filtered = append(filtered, a)
		}
	}

	createdAt := ""
	if acc.CreatedAt != nil {
		createdAt = acc.CreatedAt.Format("2006-01-02T15:04:05Z")
	}

	filtered = append(filtered, AccountRecord{
		ID:        acc.ID,
		Email:     acc.Email,
		Password:  acc.Password,
		Token:     acc.Token,
		Provider:  acc.Provider,
		CreatedAt: createdAt,
	})

	return saveAccounts(filtered)
}

// UpdateToken updates the stored token for an account in place, without
// changing its position in the list (and therefore without changing which
// account is active). Used for background token refreshes, which can happen
// on any account (e.g. `mailme inbox other@example.com`), not just the
// active one.
func UpdateToken(email, token string) error {
	accounts, err := LoadAccounts()
	if err != nil {
		return err
	}
	found := false
	for i, a := range accounts {
		if a.Email == email {
			accounts[i].Token = token
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("account not found: %s", email)
	}
	return saveAccounts(accounts)
}

// RemoveAccount removes an account by email. It returns false if the account
// wasn't found, or if it was found but persisting the removal failed.
func RemoveAccount(email string) bool {
	accounts, err := LoadAccounts()
	if err != nil {
		return false
	}
	filtered := make([]AccountRecord, 0)
	found := false
	for _, a := range accounts {
		if a.Email == email {
			found = true
			continue
		}
		filtered = append(filtered, a)
	}
	if !found {
		return false
	}
	return saveAccounts(filtered) == nil
}

// GetAccount returns an account by email, or the most recent if email is empty.
func GetAccount(email string) (*api.Account, error) {
	accounts, err := LoadAccounts()
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no saved accounts")
	}

	if email != "" {
		for _, a := range accounts {
			if a.Email == email {
				return recordToAccount(&a), nil
			}
		}
		return nil, fmt.Errorf("account not found: %s", email)
	}

	// Return most recent (last in list)
	return recordToAccount(&accounts[len(accounts)-1]), nil
}

// SetActive moves an account to the end (most recent = active). It returns
// false if the account wasn't found, or if it was found but persisting the
// change failed.
func SetActive(email string) bool {
	accounts, err := LoadAccounts()
	if err != nil {
		return false
	}
	var target *AccountRecord
	filtered := make([]AccountRecord, 0)
	for _, a := range accounts {
		if a.Email == email {
			target = &a
			continue
		}
		filtered = append(filtered, a)
	}
	if target == nil {
		return false
	}
	filtered = append(filtered, *target)
	return saveAccounts(filtered) == nil
}

func recordToAccount(r *AccountRecord) *api.Account {
	return &api.Account{
		ID:        r.ID,
		Email:     r.Email,
		Password:  r.Password,
		Token:     r.Token,
		Provider:  r.Provider,
		CreatedAt: api.ParseTime(r.CreatedAt),
	}
}
