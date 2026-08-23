package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mailme-cli/mailme/api"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	flagInterval int
	flagOnce     bool
	flagPollOnly bool
)

var watchCmd = &cobra.Command{
	Use:   "watch [email]",
	Short: "Watch for new messages in real-time",
	Long: `Watch the inbox for new messages.

Connects to real-time Mercure SSE stream or polls every INTERVAL seconds.

Examples:
  mailme watch
  mailme watch -i 3
  mailme watch --once
  mailme watch --poll`,
	Run: func(cmd *cobra.Command, args []string) {
		email := ""
		if len(args) > 0 {
			email = args[0]
		}

		client, acc, err := getActiveClientFor(email)
		if err != nil {
			fatal(err.Error())
		}

		if flagInterval < 1 {
			flagInterval = 5
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		pterm.Info.Printfln("Watching %s for new messages (Ctrl+C to stop)", pterm.Cyan(acc.Email))

		spinner, _ := pterm.DefaultSpinner.Start("Loading inbox...")
		messages, err := client.GetMessages(1)
		if err != nil {
			spinner.Fail("Failed: " + err.Error())
			return
		}
		spinner.Success()

		seenIDs := make(map[string]bool)
		for _, m := range messages {
			seenIDs[m.ID] = true
		}
		if len(messages) > 0 {
			pterm.Info.Printfln("Found %d existing message(s)", len(messages))
		}

		if flagOnce {
			return
		}

		displayNewMessage := func(m *api.Message) {
			if !seenIDs[m.ID] {
				seenIDs[m.ID] = true
				fmt.Println()
				pterm.Success.Printfln("New message! %s", pterm.Bold.Sprint(m.Subject))
				pterm.Info.Printfln("From: %s", pterm.Cyan(m.FromAddr))
				pterm.Info.Printfln("Preview: %s", truncate(m.Intro, 100))
				pterm.Info.Printfln("Read with: %s", pterm.Gray(fmt.Sprintf("mailme read %s", m.ID)))
			}
		}

		if !flagPollOnly && acc.ID != "" {
			pterm.Info.Println("Connecting to real-time Mercure stream...")
			sseErr := client.ListenMessagesSSE(ctx, acc.ID, displayNewMessage)
			if ctx.Err() != nil {
				return
			}
			pterm.Warning.Printfln("Real-time stream interrupted (%v), falling back to polling...", sseErr)
		}

		ticker := time.NewTicker(time.Duration(flagInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				messages, err := client.GetMessages(1)
				if err != nil {
					pterm.Warning.Printfln("Poll error: %s", err.Error())
					continue
				}

				for i := range messages {
					displayNewMessage(&messages[i])
				}
			}
		}
	},
}

func init() {
	watchCmd.Flags().IntVarP(&flagInterval, "interval", "i", 5, "Poll interval in seconds")
	watchCmd.Flags().BoolVar(&flagOnce, "once", false, "Check once and exit")
	watchCmd.Flags().BoolVar(&flagPollOnly, "poll", false, "Force polling instead of real-time SSE")
	rootCmd.AddCommand(watchCmd)
}
