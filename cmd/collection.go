package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/collection"
	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/platform"
	"github.com/ying-sun1/skill-tui/internal/skill"
)

func collectionStore() *collection.Store {
	home, _ := os.UserHomeDir()
	return collection.NewStore(home + "/.skill-tui")
}

var collectionCmd = &cobra.Command{
	Use:   "collection",
	Short: "Manage skill collections",
	Long:  `Create, list, and install skill collections — groups of skills that can be batch-installed.`,
}

var collectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all collections",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := collectionStore()
		collections, err := store.List()
		if err != nil {
			return err
		}

		if len(collections) == 0 {
			fmt.Println("No collections found. Create one with 'skill-tui collection create <name>'.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSKILLS\tDESCRIPTION")
		for _, c := range collections {
			fmt.Fprintf(w, "%s\t%d\t%s\n", c.Name, len(c.Skills), c.Description)
		}
		w.Flush()
		return nil
	},
}

var collectionCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new collection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")
		skills, _ := cmd.Flags().GetStringSlice("skills")

		store := collectionStore()
		if err := store.Create(name, desc, skills); err != nil {
			return err
		}

		fmt.Printf("Created collection %q with %d skill(s)\n", name, len(skills))
		return nil
	},
}

var collectionDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a collection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := collectionStore()
		if err := store.Delete(args[0]); err != nil {
			return err
		}
		fmt.Printf("Deleted collection %q\n", args[0])
		return nil
	},
}

var collectionInstallCmd = &cobra.Command{
	Use:   "install <collection-name>",
	Short: "Install all skills in a collection to all platforms",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		store := collectionStore()
		coll, err := store.Get(args[0])
		if err != nil {
			return err
		}

		registry := skill.NewRegistry(cfg.SkillsPath)
		platforms := platform.DetectInstalled(cfg)

		installed := 0
		for _, skillName := range coll.Skills {
			s, err := registry.GetSkill(skillName)
			if err != nil {
				fmt.Printf("  Skill not found: %s\n", skillName)
				continue
			}
			for _, p := range platforms {
				if p.Category == "central" || !p.Installed {
					continue
				}
				if err := platform.Install(p.SkillsDir, s.Path, s.Name); err != nil {
					fmt.Printf("  Error installing %s → %s: %v\n", s.Name, p.Name, err)
					continue
				}
				installed++
			}
		}

		fmt.Printf("Collection %q: %d installation(s) completed\n", coll.Name, installed)
		return nil
	},
}

var collectionAddCmd = &cobra.Command{
	Use:   "add <collection-name> <skill-name>",
	Short: "Add a skill to a collection",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := collectionStore()
		if err := store.AddSkill(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Added %q to collection %q\n", args[1], args[0])
		return nil
	},
}

var collectionRemoveCmd = &cobra.Command{
	Use:   "remove <collection-name> <skill-name>",
	Short: "Remove a skill from a collection",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := collectionStore()
		if err := store.RemoveSkill(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Removed %q from collection %q\n", args[1], args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(collectionCmd)
	collectionCmd.AddCommand(collectionListCmd)
	collectionCmd.AddCommand(collectionCreateCmd)
	collectionCmd.AddCommand(collectionDeleteCmd)
	collectionCmd.AddCommand(collectionInstallCmd)
	collectionCmd.AddCommand(collectionAddCmd)
	collectionCmd.AddCommand(collectionRemoveCmd)

	collectionCreateCmd.Flags().StringP("description", "d", "", "Collection description")
	collectionCreateCmd.Flags().StringSlice("skills", nil, "Initial skills to include")
}
