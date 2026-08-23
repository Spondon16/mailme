package cmd

import (
	"github.com/Spondon16/mailme/config"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var flagForce bool

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a message or account",
}

var deleteMessageCmd = &cobra.Command{
	Use:   "message <message-id> [email]",
	Short: "Delete a specific message",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		messageID := args[0]
		email := ""
		if len(args) > 1 {
			email = args[1]
		}

		if !flagForce {
			confirm, err := pterm.DefaultInteractiveConfirm.
				WithDefaultText("Delete message " + messageID + "?").
				Show()
			if err != nil || !confirm {
				return
			}
		}

		client, _, err := getActiveClientFor(email)
		if err != nil {
			fatal(err.Error())
		}

		spinner, _ := pterm.DefaultSpinner.Start("Deleting message...")
		if err := client.DeleteMessage(messageID); err != nil {
			spinner.Fail("Failed: " + err.Error())
			return
		}
		spinner.Success()
		pterm.Success.Printfln("Message %s deleted.", messageID)
	},
}

var deleteAccountCmd = &cobra.Command{
	Use:   "account [email]",
	Short: "Delete the current or specified account",
	Run: func(cmd *cobra.Command, args []string) {
		email := ""
		if len(args) > 0 {
			email = args[0]
		}

		client, acc, err := getActiveClientFor(email)
		if err != nil {
			fatal(err.Error())
		}

		if !flagForce {
			confirm, err := pterm.DefaultInteractiveConfirm.
				WithDefaultText("Delete account " + acc.Email + "? This cannot be undone.").
				Show()
			if err != nil || !confirm {
				return
			}
		}

		spinner, _ := pterm.DefaultSpinner.Start("Deleting account...")
		if err := client.DeleteAccount(acc.ID); err != nil {
			printWarning("Remote delete failed (removing locally): " + err.Error())
		}
		config.RemoveAccount(acc.Email)
		spinner.Success()
		pterm.Success.Printfln("Account %s deleted.", acc.Email)
	},
}

func init() {
	deleteMessageCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Skip confirmation")
	deleteAccountCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Skip confirmation")
	deleteCmd.AddCommand(deleteMessageCmd)
	deleteCmd.AddCommand(deleteAccountCmd)
	rootCmd.AddCommand(deleteCmd)
}
