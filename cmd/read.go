package cmd

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	flagRaw  bool
	flagHTML bool
	flagEML  bool
)

var readCmd = &cobra.Command{
	Use:   "read <message-id> [email]",
	Short: "Read a specific email message",
	Long: `Read a specific email message by its ID.

MESSAGE_ID can be found via 'mailme inbox'.

Examples:
  mailme read 12345
  mailme read 12345 user@example.com
  mailme read 12345 --raw
  mailme read 12345 --html
  mailme read 12345 --eml`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		messageID := args[0]
		email := ""
		if len(args) > 1 {
			email = args[1]
		}

		client, _, err := getActiveClientFor(email)
		if err != nil {
			fatal(err.Error())
		}

		if flagEML {
			spinner, _ := pterm.DefaultSpinner.Start("Fetching message source...")
			source, err := client.GetSource(messageID)
			if err != nil {
				spinner.Fail("Failed: " + err.Error())
				return
			}
			spinner.Success()
			_ = client.SetSeen(messageID, true)
			fmt.Println(source)
			return
		}

		spinner, _ := pterm.DefaultSpinner.Start("Fetching message...")
		msg, err := client.GetMessage(messageID)
		if err != nil {
			spinner.Fail("Failed: " + err.Error())
			return
		}
		spinner.Success()

		// Automatically mark message as read
		if !msg.Seen {
			_ = client.SetSeen(messageID, true)
		}

		if flagRaw {
			fmt.Println(msg.Text)
			return
		}
		if flagHTML {
			fmt.Println(msg.HTML)
			return
		}

		fmt.Println()
		pterm.DefaultSection.WithLevel(2).Println("Message Headers")
		pterm.Info.Printfln("From:    %s", pterm.Cyan(msg.FromAddr))
		pterm.Info.Printfln("To:      %s", msg.To)
		if msg.Cc != "" {
			pterm.Info.Printfln("Cc:      %s", msg.Cc)
		}
		if msg.Bcc != "" {
			pterm.Info.Printfln("Bcc:     %s", msg.Bcc)
		}
		pterm.Info.Printfln("Subject: %s", pterm.Bold.Sprint(msg.Subject))
		if msg.CreatedAt != nil {
			pterm.Info.Printfln("Date:    %s", msg.CreatedAt.Format("2006-01-02 15:04:05"))
		}

		fmt.Println()
		pterm.DefaultSection.WithLevel(2).Println("Body")
		body := msg.Text
		if body == "" {
			body = msg.Intro
		}
		if body == "" {
			body = "(no content)"
		}
		fmt.Println(body)

		if len(msg.Attachments) > 0 {
			fmt.Println()
			pterm.DefaultSection.WithLevel(2).Println("Attachments")
			data := pterm.TableData{
				{"ID", "Filename", "Type", "Size"},
			}
			for _, a := range msg.Attachments {
				data = append(data, []string{a.ID, a.Filename, a.ContentType, formatSize(a.Size)})
			}
			t, _ := pterm.DefaultTable.
				WithHasHeader().
				WithHeaderRowSeparator("-").
				WithData(data).
				Srender()
			fmt.Println(t)
		}
	},
}

func formatSize(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(bytes)
	for _, u := range units {
		if size < 1024 {
			return fmt.Sprintf("%.1f %s", size, u)
		}
		size /= 1024
	}
	return fmt.Sprintf("%.1f TB", size)
}

func init() {
	readCmd.Flags().BoolVar(&flagRaw, "raw", false, "Show raw text content only")
	readCmd.Flags().BoolVar(&flagHTML, "html", false, "Show HTML content")
	readCmd.Flags().BoolVar(&flagEML, "eml", false, "Show raw RFC 822 / EML source")
	rootCmd.AddCommand(readCmd)
}
