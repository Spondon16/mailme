package cmd

import (
	"fmt"
	"os"

	"github.com/Spondon16/mailme/api"
	"github.com/Spondon16/mailme/config"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var version = "1.0.0"

var rootCmd = &cobra.Command{
	Use:     "mailme",
	Short:   "Powerful CLI temporary email tool",
	Version: version,
	Long: `mailme — disposable email from your terminal

Generate, read, and manage temporary email addresses
without leaving the command line.

Quick Start:
  mailme g              Generate a new email (auto-copies to clipboard)
  mailme m              Check your inbox
  mailme read <id>      Read a specific message
  mailme me             Show current account
  mailme d              Delete current account

Documentation:
  mailme <command> --help    Show help for any command
  mailme <command> -h        Same thing, shorter`,
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetHelpTemplate(rootHelpTemplate)
	rootCmd.SetUsageTemplate(rootUsageTemplate)
}

// --- Custom Templates ---

var rootUsageTemplate = `{{.Short}}

Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}

Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

var rootHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

// --- Provider Helpers ---

// newClient returns the default (mail.tm) provider client.
func newClient() api.Provider {
	client, err := api.NewProvider("")
	if err != nil {
		fatal(err.Error())
	}
	return client
}

// getActiveClient returns a client configured with the active account's credentials.
func getActiveClient() (api.Provider, *api.Account, error) {
	acc, err := config.GetAccount("")
	if err != nil {
		return nil, nil, err
	}
	return clientForAccount(acc), acc, nil
}

// getActiveClientFor returns a client for a specific email (or active if empty).
func getActiveClientFor(email string) (api.Provider, *api.Account, error) {
	acc, err := config.GetAccount(email)
	if err != nil {
		return nil, nil, err
	}
	return clientForAccount(acc), acc, nil
}

// clientForAccount builds a fully configured client with token + credentials + refresh callback.
func clientForAccount(acc *api.Account) api.Provider {
	client, err := api.NewProvider(acc.Provider)
	if err != nil {
		// Unrecognized/legacy provider name — fall back to the default.
		client = newClient()
	}
	if acc.Token != "" {
		client.SetToken(acc.Token)
	}
	// Pass credentials for automatic token refresh on 401
	client.SetCredentials(acc.Email, acc.Password)
	// Persist refreshed token to disk, without disturbing which account is
	// active — a background refresh can happen on any account, not just the
	// active one (e.g. `mailme inbox other@example.com`).
	client.OnTokenUpdate(func(newToken string) {
		acc.Token = newToken
		if err := config.UpdateToken(acc.Email, newToken); err != nil {
			printWarning("Failed to persist refreshed token: " + err.Error())
		}
	})
	return client
}

func printSuccess(msg string) {
	pterm.Success.Println(msg)
}

func printError(msg string) {
	pterm.Error.Println(msg)
}

func printInfo(msg string) {
	pterm.Info.Println(msg)
}

func printWarning(msg string) {
	pterm.Warning.Println(msg)
}

func fatal(msg string) {
	pterm.Error.Println(msg)
	os.Exit(1)
}

func fatalf(format string, args ...interface{}) {
	pterm.Error.Println(fmt.Sprintf(format, args...))
	os.Exit(1)
}
