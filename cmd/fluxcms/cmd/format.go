package cmd

import (
	"fmt"
	"os"

	"github.com/infrasutra/fsl/parser"
	"github.com/spf13/cobra"
)

var formatCheck bool

var formatCmd = &cobra.Command{
	Use:   "format [file/dir]",
	Short: "Format FSL schema files to canonical style",
	Long: `Format one or more FSL schema files using canonical FSL style.

Examples:
  fluxcms format schema.fsl
  fluxcms format ./schemas
  fluxcms format --check ./schemas`,
	RunE: runFormat,
}

func init() {
	rootCmd.AddCommand(formatCmd)
	formatCmd.Flags().BoolVar(&formatCheck, "check", false, "Check if files are already formatted without writing changes")
}

func runFormat(cmd *cobra.Command, args []string) error {
	paths := args
	if len(paths) == 0 {
		paths = []string{GetSchemaDirectory()}
	}

	files, err := collectFSLFiles(paths)
	if err != nil {
		return err
	}

	changed := 0
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", file, err)
		}

		formatted, err := parser.Format(string(content))
		if err != nil {
			return fmt.Errorf("cannot format %s: %w", file, err)
		}

		if formatted == string(content) {
			continue
		}

		changed++
		if formatCheck {
			fmt.Printf("would format %s\n", file)
			continue
		}

		info, err := os.Stat(file)
		if err != nil {
			return fmt.Errorf("cannot stat %s: %w", file, err)
		}

		if err := os.WriteFile(file, []byte(formatted), info.Mode().Perm()); err != nil {
			return fmt.Errorf("cannot write %s: %w", file, err)
		}
		fmt.Printf("formatted %s\n", file)
	}

	if formatCheck {
		if changed > 0 {
			return fmt.Errorf("%d file(s) need formatting", changed)
		}
		fmt.Printf("all %d file(s) already formatted\n", len(files))
		return nil
	}

	if changed == 0 {
		fmt.Printf("all %d file(s) already formatted\n", len(files))
		return nil
	}

	fmt.Printf("formatted %d file(s)\n", changed)
	return nil
}
