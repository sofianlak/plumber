package cmd

import (
	"fmt"
	"os"

	"github.com/getplumber/plumber/internal/defaultconfig"
	"github.com/spf13/cobra"
)

var (
	generateOutputFile string
	forceOutput        bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Plumber resources",
	Long:  `Generate Plumber resources like configuration files.`,
}

var generateConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Generate a default .plumber.yaml configuration file",
	Long: `Generate a default .plumber.yaml configuration file.

This creates a configuration file with sensible defaults that you can
customize for your organization's compliance requirements.

The generated config includes:
- Container image tag policies (forbid 'latest', 'dev', etc.)
- Trusted registry whitelist
- Branch protection requirements

Examples:
  # Generate default config in current directory
  plumber generate config

  # Generate config with custom filename
  plumber generate config --output my-plumber-config.yaml

  # Overwrite existing file
  plumber generate config --force
`,
	RunE: runGenerateConfig,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generateConfigCmd)

	generateConfigCmd.Flags().StringVarP(&generateOutputFile, "output", "o", ".plumber.yaml", "Output file path")
	generateConfigCmd.Flags().BoolVarP(&forceOutput, "force", "f", false, "Overwrite existing file")
}

func runGenerateConfig(cmd *cobra.Command, args []string) error {
	// Check if file already exists
	if _, err := os.Stat(generateOutputFile); err == nil {
		if !forceOutput {
			return fmt.Errorf("file %s already exists. Use --force to overwrite", generateOutputFile)
		}
		fmt.Fprintf(os.Stderr, "Overwriting existing file: %s\n", generateOutputFile)
	}

	// Get embedded default config
	configContent := defaultconfig.Get()

	// Write to file
	if err := os.WriteFile(generateOutputFile, configContent, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Generated %s\n", generateOutputFile)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review and customize the configuration for your needs")
	fmt.Println("  2. Export the GITLAB_TOKEN environment variable if you haven't already")
	fmt.Println("  3. Run: plumber analyze --gitlab-url <url> --project <path>")

	return nil
}
