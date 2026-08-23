package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	flagJSON bool
	flagPage int
)

var inboxCmd = &cobra.Command{
	Use:     "inbox [email]",
	Aliases: []string{"m"},
	Short:   "List messages in the inbox",
	Long: `List all messages in the inbox.

Examples:
  mailme inbox
  mailme inbox user@example.com
  mailme inbox --page 2
  mailme inbox --json`,
	Run: func(cmd *cobra.Command, args []string) {
		email := ""
		if len(args) > 0 {
			email = args[0]
		}

		client, _, err := getActiveClientFor(email)
		if err != nil {
			fatal(err.Error())
		}

		if flagPage < 1 {
			flagPage = 1
		}

		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Fetching messages (page %d)...", flagPage))
		messages, err := client.GetMessages(flagPage)
		if err != nil {
			spinner.Fail("Failed: " + err.Error())
			return
		}
		spinner.Success()

		if flagJSON {
			output := make([]map[string]interface{}, 0)
			for _, m := range messages {
				output = append(output, map[string]interface{}{
					"id":      m.ID,
					"subject": m.Subject,
					"from":    m.FromAddr,
					"intro":   m.Intro,
					"seen":    m.Seen,
				})
			}
			data, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(data))
			return
		}

		if len(messages) == 0 {
			if flagPage > 1 {
				pterm.Info.Printfln("No messages found on page %d.", flagPage)
			} else {
				pterm.Info.Println("No messages.")
			}
			return
		}

		data := pterm.TableData{
			{"ID", "Status", "From", "Subject", "Preview"},
		}
		for _, m := range messages {
			status := pterm.Green("new")
			if m.Seen {
				status = pterm.Gray("read")
			}
			data = append(data, []string{
				m.ID,
				status,
				truncate(m.FromAddr, 25),
				truncate(m.Subject, 30),
				truncate(m.Intro, 35),
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

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}

func init() {
	inboxCmd.Flags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	inboxCmd.Flags().IntVarP(&flagPage, "page", "p", 1, "Page number to view")
	rootCmd.AddCommand(inboxCmd)
}
