package cmd

import (
	"fmt"
	"strings"

	"github.com/Spondon16/mailme/api"
	"github.com/Spondon16/mailme/config"
	"github.com/Spondon16/mailme/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	flagUsername string
	flagPassword string
	flagDomain   string
	flagProvider string
)

var generateCmd = &cobra.Command{
	Use:     "generate [email]",
	Aliases: []string{"g"},
	Short:   "Generate a new temporary email address",
	Long: `Generate a new temporary email address.

If an email address is provided, it will be used as-is.
Otherwise, a random username is generated.

Examples:
  mailme generate
  mailme generate -u myname
  mailme generate user@example.com -w mypassword
  mailme generate -d example.com
  mailme generate -p tempmail.plus`,
	Run: func(cmd *cobra.Command, args []string) {
		emailAddr := ""
		if len(args) > 0 {
			emailAddr = strings.TrimSpace(args[0])
		}

		username := flagUsername
		password := flagPassword
		domain := flagDomain
		provider := strings.ToLower(strings.TrimSpace(flagProvider))

		if emailAddr != "" {
			if strings.Contains(emailAddr, "@") {
				parts := strings.SplitN(emailAddr, "@", 2)
				username = parts[0]
				domain = parts[1]
			} else if username == "" {
				username = emailAddr
			}
		}

		if password == "" {
			password = utils.RandomPassword()
		}

		client, err := api.NewProvider(provider)
		if err != nil {
			fatal(err.Error())
		}

		// Third-party providers (see api.ThirdPartyClient) always assign a
		// full random address themselves — no custom username or domain.
		if provider != "" && provider != "mailtm" {
			if username != "" || domain != "" {
				printWarning(fmt.Sprintf("%s does not support choosing a username or domain; generating a random address instead.", provider))
			}

			spinner, _ := pterm.DefaultSpinner.Start("Creating account...")
			account, err := client.CreateAccount("", password)
			if err != nil {
				spinner.Fail("Failed to create account: " + err.Error())
				return
			}
			spinner.Success()

			account.Provider = provider
			saveAndAnnounce(account)
			return
		}

		if username == "" {
			username = utils.RandomUsername()
		}

		if domain == "" {
			spinner, _ := pterm.DefaultSpinner.Start("Fetching domains...")
			domains, err := client.GetDomains()
			if err != nil {
				spinner.Fail("Failed to fetch domains: " + err.Error())
				return
			}
			spinner.Success()
			if len(domains) == 0 {
				fatal("No domains available")
			}
			domain = domains[0].Name
		}

		email := username + "@" + domain

		spinner, _ := pterm.DefaultSpinner.Start("Creating account...")
		account, err := client.CreateAccount(email, password)
		if err != nil {
			spinner.Fail("Failed to create account: " + err.Error())
			return
		}
		spinner.Success()

		account.Provider = "mailtm"
		saveAndAnnounce(account)
	},
}

// saveAndAnnounce persists a newly generated account, makes it active,
// copies its address to the clipboard, and prints it.
func saveAndAnnounce(account *api.Account) {
	if err := config.AddAccount(account); err != nil {
		printWarning("Account created but failed to save locally: " + err.Error())
	}
	config.SetActive(account.Email)

	fmt.Println()
	pterm.DefaultSection.WithLevel(2).Printfln("Email: %s", pterm.Cyan(account.Email))
	fmt.Println()

	if err := utils.CopyToClipboard(account.Email); err != nil {
		printWarning("Could not copy to clipboard: " + err.Error())
	} else {
		printSuccess("Copied to clipboard!")
	}
}

func init() {
	generateCmd.Flags().StringVarP(&flagUsername, "username", "u", "", "Username for the email")
	generateCmd.Flags().StringVarP(&flagPassword, "password", "w", "", "Password for the account")
	generateCmd.Flags().StringVarP(&flagDomain, "domain", "d", "", "Email domain (random if omitted)")
	generateCmd.Flags().StringVarP(&flagProvider, "provider", "p", "mailtm", "Provider to use (mailtm, tempmail.plus, tempmailc, mailnesia)")
	rootCmd.AddCommand(generateCmd)
}
