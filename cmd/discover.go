package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/discover"
)

var discoverCmd = &cobra.Command{
	Use:   "discover [path]",
	Short: "Discover local project-level skills",
	Long:  `Scan a directory for project-level skills (.claude/skills/, .agents/skills/, etc.) and list them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		scanPath := "."
		if len(args) > 0 {
			scanPath = args[0]
		}

		info, err := os.Stat(scanPath)
		if err != nil {
			return fmt.Errorf("path not found: %s", scanPath)
		}
		if !info.IsDir() {
			return fmt.Errorf("not a directory: %s", scanPath)
		}

		recursive, _ := cmd.Flags().GetBool("recursive")

		var discoveries []discover.Discovery
		if recursive {
			discoveries, err = discover.ScanRecursive(scanPath, 5)
		} else {
			discoveries, err = discover.Scan(scanPath)
		}
		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}

		if len(discoveries) == 0 {
			fmt.Printf("No project-level skills found in %s\n", scanPath)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPLATFORM\tPATH")
		for _, d := range discoveries {
			fmt.Fprintf(w, "%s\t%s\t%s\n", d.Name, d.Platform, d.Path)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(discoverCmd)
	discoverCmd.Flags().BoolP("recursive", "r", false, "Scan subdirectories recursively")
}
