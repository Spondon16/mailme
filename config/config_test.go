package config

import (
	"path/filepath"
	"testing"

	"github.com/Spondon16/mailme/api"
)

// useTempConfigDir points the package-level config paths at a fresh temp
// directory for the duration of the test, so tests never touch the real
// ~/.config/mailme.
func useTempConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	origDir, origFile := configDir, accountsFile
	configDir = dir
	accountsFile = filepath.Join(dir, "accounts.json")
	t.Cleanup(func() {
		configDir, accountsFile = origDir, origFile
	})
}

func TestLoadAccountsEmpty(t *testing.T) {
	useTempConfigDir(t)

	accounts, err := LoadAccounts()
	if err != nil {
		t.Fatalf("LoadAccounts() error = %v, want nil", err)
	}
	if len(accounts) != 0 {
		t.Fatalf("LoadAccounts() = %v, want empty", accounts)
	}
}

func TestAddAndGetAccount(t *testing.T) {
	useTempConfigDir(t)

	acc := &api.Account{ID: "1", Email: "a@example.com", Password: "pw", Provider: "mailtm"}
	if err := AddAccount(acc); err != nil {
		t.Fatalf("AddAccount() error = %v", err)
	}

	got, err := GetAccount("a@example.com")
	if err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}
	if got.Email != acc.Email || got.Provider != acc.Provider {
		t.Fatalf("GetAccount() = %+v, want matching %+v", got, acc)
	}
}

func TestAddAccountReplacesExisting(t *testing.T) {
	useTempConfigDir(t)

	_ = AddAccount(&api.Account{ID: "1", Email: "a@example.com", Provider: "mailtm"})
	_ = AddAccount(&api.Account{ID: "2", Email: "a@example.com", Provider: "tempmailc"})

	accounts, err := LoadAccounts()
	if err != nil {
		t.Fatalf("LoadAccounts() error = %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account after re-adding same email, got %d", len(accounts))
	}
	if accounts[0].Provider != "tempmailc" {
		t.Fatalf("expected re-added account to overwrite provider, got %q", accounts[0].Provider)
	}
}

func TestGetAccountReturnsMostRecentWhenEmailEmpty(t *testing.T) {
	useTempConfigDir(t)

	_ = AddAccount(&api.Account{ID: "1", Email: "first@example.com", Provider: "mailtm"})
	_ = AddAccount(&api.Account{ID: "2", Email: "second@example.com", Provider: "mailtm"})

	got, err := GetAccount("")
	if err != nil {
		t.Fatalf("GetAccount(\"\") error = %v", err)
	}
	if got.Email != "second@example.com" {
		t.Fatalf("GetAccount(\"\") = %q, want most recently added account", got.Email)
	}
}

func TestGetAccountNotFound(t *testing.T) {
	useTempConfigDir(t)

	if _, err := GetAccount("missing@example.com"); err == nil {
		t.Fatal("GetAccount() error = nil, want error for unknown email")
	}
}

func TestUpdateToken(t *testing.T) {
	useTempConfigDir(t)

	_ = AddAccount(&api.Account{ID: "1", Email: "a@example.com", Token: "old", Provider: "mailtm"})

	if err := UpdateToken("a@example.com", "new"); err != nil {
		t.Fatalf("UpdateToken() error = %v", err)
	}

	got, err := GetAccount("a@example.com")
	if err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}
	if got.Token != "new" {
		t.Fatalf("Token = %q, want %q", got.Token, "new")
	}
}

func TestUpdateTokenUnknownAccount(t *testing.T) {
	useTempConfigDir(t)

	if err := UpdateToken("missing@example.com", "tok"); err == nil {
		t.Fatal("UpdateToken() error = nil, want error for unknown account")
	}
}

func TestRemoveAccount(t *testing.T) {
	useTempConfigDir(t)

	_ = AddAccount(&api.Account{ID: "1", Email: "a@example.com", Provider: "mailtm"})

	if !RemoveAccount("a@example.com") {
		t.Fatal("RemoveAccount() = false, want true")
	}
	if RemoveAccount("a@example.com") {
		t.Fatal("RemoveAccount() on already-removed account = true, want false")
	}

	accounts, _ := LoadAccounts()
	if len(accounts) != 0 {
		t.Fatalf("expected no accounts left, got %d", len(accounts))
	}
}

func TestSetActiveMovesAccountToEnd(t *testing.T) {
	useTempConfigDir(t)

	_ = AddAccount(&api.Account{ID: "1", Email: "first@example.com", Provider: "mailtm"})
	_ = AddAccount(&api.Account{ID: "2", Email: "second@example.com", Provider: "mailtm"})

	if !SetActive("first@example.com") {
		t.Fatal("SetActive() = false, want true")
	}

	got, err := GetAccount("")
	if err != nil {
		t.Fatalf("GetAccount(\"\") error = %v", err)
	}
	if got.Email != "first@example.com" {
		t.Fatalf("active account = %q, want %q", got.Email, "first@example.com")
	}
}

func TestSetActiveUnknownAccount(t *testing.T) {
	useTempConfigDir(t)

	if SetActive("missing@example.com") {
		t.Fatal("SetActive() = true, want false for unknown account")
	}
}
