package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	flagOutput      string
	flagDownloadEML bool
)

var downloadCmd = &cobra.Command{
	Use:   "download <message-id> [attachment-id] [email]",
	Short: "Download an email attachment or raw .eml message",
	Long: `Download an attachment from a specific message, or download the raw .eml message.

MESSAGE_ID and ATTACHMENT_ID can be found via 'mailme read MESSAGE_ID'.

Examples:
  mailme download 12345 att_abc123
  mailme download 12345 att_abc123 -o ~/Downloads
  mailme download 12345 --eml
  mailme download 12345 -o message.eml`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		messageID := args[0]
		attachmentID := ""
		email := ""

		if len(args) > 1 {
			// If second arg contains "@", it's email, otherwise attachmentID
			if strings.Contains(args[1], "@") {
				email = args[1]
			} else {
				attachmentID = args[1]
				if len(args) > 2 {
					email = args[2]
				}
			}
		}

		client, _, err := getActiveClientFor(email)
		if err != nil {
			fatal(err.Error())
		}

		var data []byte
		var defaultFilename string

		if flagDownloadEML || attachmentID == "" {
			// Download full EML
			spinner, _ := pterm.DefaultSpinner.Start("Downloading message (.eml)...")
			emlData, err := client.DownloadMessageEML(messageID)
			if err != nil {
				spinner.Fail("Failed to download message: " + err.Error())
				return
			}
			spinner.Success()
			data = emlData
			defaultFilename = fmt.Sprintf("%s.eml", messageID)
		} else {
			// Download attachment
			spinner, _ := pterm.DefaultSpinner.Start("Fetching attachment info...")
			msg, err := client.GetMessage(messageID)
			if err != nil {
				spinner.Fail("Failed to fetch message: " + err.Error())
				return
			}

			filename := attachmentID
			for _, att := range msg.Attachments {
				if att.ID == attachmentID {
					filename = att.Filename
					break
				}
			}
			defaultFilename = filename
			spinner.UpdateText("Downloading " + filename + "...")

			attData, err := client.DownloadAttachment(messageID, attachmentID)
			if err != nil {
				spinner.Fail("Failed: " + err.Error())
				return
			}
			spinner.Success()
			data = attData
		}

		// Determine destination path
		outPath := flagOutput
		fileInfo, err := os.Stat(outPath)
		if err == nil && fileInfo.IsDir() {
			outPath = filepath.Join(outPath, filepath.Base(defaultFilename))
		} else if strings.HasSuffix(outPath, "/") || strings.HasSuffix(outPath, "\\") {
			if err := os.MkdirAll(outPath, 0755); err != nil {
				printError("Failed to create directory: " + err.Error())
				return
			}
			outPath = filepath.Join(outPath, filepath.Base(defaultFilename))
		} else {
			parentDir := filepath.Dir(outPath)
			if parentDir != "" && parentDir != "." {
				if err := os.MkdirAll(parentDir, 0755); err != nil {
					printError("Failed to create directory: " + err.Error())
					return
				}
			}
		}

		// Atomic write for downloaded file
		tmpPath := outPath + fmt.Sprintf(".tmp.%d", time.Now().UnixNano())
		if err := os.WriteFile(tmpPath, data, 0644); err != nil {
			printError("Failed to write file: " + err.Error())
			return
		}

		if err := os.Rename(tmpPath, outPath); err != nil {
			_ = os.Remove(tmpPath)
			printError("Failed to save downloaded file: " + err.Error())
			return
		}

		pterm.Success.Printfln("Downloaded to %s (%d bytes)", pterm.Cyan(outPath), len(data))
	},
}

func init() {
	downloadCmd.Flags().StringVarP(&flagOutput, "output", "o", ".", "Output directory or file path")
	downloadCmd.Flags().BoolVar(&flagDownloadEML, "eml", false, "Download raw .eml file")
	rootCmd.AddCommand(downloadCmd)
}
