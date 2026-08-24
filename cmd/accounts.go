package cmd

import (
	"fmt"

	"github.com/Spondon16/mailme/config"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage saved email accounts",
	Long: `Manage saved email accounts.

Without subcommands, lists all saved accounts.

Subcommands:
  mailme accounts show [email]       Show account details
  mailme accounts use <email>        Switch active account
  mailme accounts delete <email>     Delete an account
  mailme accounts prune              Remove all local accounts`,
	Run: func(cmd *cobra.Command, args []string) {
		accounts, err := config.LoadAccounts()
		if err != nil {
			fatal(err.Error())
		}
		if len(accounts) == 0 {
			pterm.Info.Println("No saved accounts. Run `mailme generate` to create one.")
			return
		}

		active, _ := config.GetAccount("")

		data := pterm.TableData{
			{"#", "Email", "Provider", "Active"},
		}
		for i, a := range accounts {
			activeMark := ""
			if active != nil && a.Email == active.Email {
				activeMark = pterm.Green("yes")
			}
			data = append(data, []string{
				fmt.Sprintf("%d", i+1),
				a.Email,
				a.Provider,
				activeMark,
			})
		}

		t, _ := pterm.DefaultTable.
			WithHasHeader().
			WithHeaderRowSeparator("-").
			WithBoxed(true).
			WithData(data).
			Srender()
		fmt.Println(t)
	},
}

var accountsUseCmd = &cobra.Command{
	Use:   "use <email>",
	Short: "Switch to a different saved account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if config.SetActive(args[0]) {
			pterm.Success.Printfln("Switched to %s", pterm.Cyan(args[0]))
		} else {
			pterm.Error.Printfln("Account %s not found.", args[0])
		}
	},
}

var accountsShowCmd = &cobra.Command{
	Use:   "show [email]",
	Short: "Show details of a saved account",
	Run: func(cmd *cobra.Command, args []string) {
		email := ""
		if len(args) > 0 {
			email = args[0]
		}
		showAccount(email)
	},
}

var accountsDeleteCmd = &cobra.Command{
	Use:   "delete <email>",
	Short: "Delete an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		email := args[0]

		if !flagForce {
			confirm, err := pterm.DefaultInteractiveConfirm.
				WithDefaultText("Delete " + email + "? This cannot be undone.").
				Show()
			if err != nil || !confirm {
				return
			}
		}

		acc, err := config.GetAccount(email)
		if err != nil {
			fatal(err.Error())
		}

		client, _, err := getActiveClientFor(email)
		if err != nil {
			fatal(err.Error())
		}

		spinner, _ := pterm.DefaultSpinner.Start("Deleting " + email)
		if err := client.DeleteAccount(acc.ID); err != nil {
			printWarning("Remote delete failed (removing locally): " + err.Error())
		}
		if !config.RemoveAccount(email) {
			spinner.Fail("Failed to remove account locally.")
			return
		}
		spinner.Success()
		pterm.Success.Printfln("Deleted %s", email)
	},
}

var accountsPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove all local accounts",
	Long: `Remove all saved accounts from local storage.

This does NOT delete accounts from the remote server —
it only clears the local config file.`,
	Run: func(cmd *cobra.Command, args []string) {
		accounts, err := config.LoadAccounts()
		if err != nil {
			fatal(err.Error())
		}
		if len(accounts) == 0 {
			pterm.Info.Println("No accounts to prune.")
			return
		}

		if !flagForce {
			confirm, err := pterm.DefaultInteractiveConfirm.
				WithDefaultText(fmt.Sprintf("Remove all %d saved accounts locally?", len(accounts))).
				Show()
			if err != nil || !confirm {
				return
			}
		}

		failed := 0
		for _, a := range accounts {
			if !config.RemoveAccount(a.Email) {
				failed++
			}
		}
		if failed > 0 {
			printWarning(fmt.Sprintf("Failed to remove %d of %d accounts.", failed, len(accounts)))
		}
		pterm.Success.Printfln("Pruned %d accounts.", len(accounts)-failed)
	},
}

func init() {
	accountsCmd.AddCommand(accountsUseCmd)
	accountsCmd.AddCommand(accountsShowCmd)
	accountsCmd.AddCommand(accountsDeleteCmd)
	accountsCmd.AddCommand(accountsPruneCmd)
	accountsDeleteCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Skip confirmation")
	accountsPruneCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Skip confirmation")
	rootCmd.AddCommand(accountsCmd)
	rootCmd.AddCommand(meCmd)
	rootCmd.AddCommand(dCmd)
	dCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Skip confirmation")
}

func showAccount(email string) {
	acc, err := config.GetAccount(email)
	if err != nil {
		fatal(err.Error())
	}

	fmt.Println()
	pterm.DefaultSection.WithLevel(2).Println("Account")
	pterm.Info.Printfln("Email:    %s", pterm.Cyan(acc.Email))
	pterm.Info.Printfln("Provider: %s", acc.Provider)
	if acc.CreatedAt != nil {
		pterm.Info.Printfln("Created:  %s", acc.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()
}

// me is a top-level shorthand for "accounts show"
var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show current account details",
	Run: func(cmd *cobra.Command, args []string) {
		acc, err := config.GetAccount("")
		if err != nil {
			fatal(err.Error())
		}

		fmt.Println()
		pterm.DefaultSection.WithLevel(2).Println("Account")
		pterm.Info.Printfln("Email:    %s", pterm.Cyan(acc.Email))
		pterm.Info.Printfln("Provider: %s", acc.Provider)
		if acc.CreatedAt != nil {
			pterm.Info.Printfln("Created:  %s", acc.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	},
}

// d is a top-level shorthand for "delete account"
var dCmd = &cobra.Command{
	Use:   "d",
	Short: "Delete current account",
	Run: func(cmd *cobra.Command, args []string) {
		acc, err := config.GetAccount("")
		if err != nil {
			fatal(err.Error())
		}

		if !flagForce {
			confirm, err := pterm.DefaultInteractiveConfirm.
				WithDefaultText("Delete " + acc.Email + "?").
				Show()
			if err != nil || !confirm {
				return
			}
		}

		client, _, err := getActiveClientFor("")
		if err != nil {
			fatal(err.Error())
		}

		spinner, _ := pterm.DefaultSpinner.Start("Deleting...")
		if err := client.DeleteAccount(acc.ID); err != nil {
			printWarning("Remote delete failed (removing locally): " + err.Error())
		}
		if !config.RemoveAccount(acc.Email) {
			spinner.Fail("Failed to remove account locally.")
			return
		}
		spinner.Success()
		pterm.Success.Printfln("Deleted %s", acc.Email)
	},
}
