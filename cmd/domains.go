package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var domainsJSON bool

var domainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "List available email domains",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		spinner, _ := pterm.DefaultSpinner.Start("Fetching domains...")
		domains, err := client.GetDomains()
		if err != nil {
			spinner.Fail("Failed: " + err.Error())
			return
		}
		spinner.Success()

		if len(domains) == 0 {
			pterm.Info.Println("No domains available.")
			return
		}

		if domainsJSON {
			names := make([]string, 0, len(domains))
			for _, d := range domains {
				names = append(names, d.Name)
			}
			data, _ := json.MarshalIndent(names, "", "  ")
			fmt.Println(string(data))
			return
		}

		pterm.DefaultSection.WithLevel(2).Println("Available domains")
		for _, d := range domains {
			pterm.Info.Printfln("  %s", pterm.Cyan(d.Name))
		}
	},
}

func init() {
	domainsCmd.Flags().BoolVar(&domainsJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(domainsCmd)
}
