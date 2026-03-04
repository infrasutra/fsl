package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Version information (set at build time)
var (
	Version   = "dev"
	BuildDate = "unknown"
)

type Config struct {
	Version   string `yaml:"version"`
	Workspace struct {
		APIURL string `yaml:"api_url"`
		APIKey string `yaml:"api_key"`
	} `yaml:"workspace"`
	Schemas struct {
		Directory string `yaml:"directory"`
	} `yaml:"schemas"`
	Output struct {
		TypeScript struct {
			Directory string `yaml:"directory"`
			Client    string `yaml:"client"`
		} `yaml:"typescript"`
		Python struct {
			Directory string `yaml:"directory"`
		} `yaml:"python"`
	} `yaml:"output"`
	Lint struct {
		NamingConvention      *bool `yaml:"naming_convention"`
		UnusedTypes           *bool `yaml:"unused_types"`
		RequiredFieldOrdering *bool `yaml:"required_field_ordering"`
		RelationCardinality   *bool `yaml:"relation_cardinality"`
		MaxFieldCount         int   `yaml:"max_field_count"`
	} `yaml:"lint"`
}

var (
	cfgFile string
	config  *Config
)

var rootCmd = &cobra.Command{
	Use:   "fsl",
	Short: "FSL CLI - Schema-first headless CMS tooling",
	Long: `FSL CLI provides tools for managing FSL schemas, migrations,
TypeScript SDK generation, and CMS server integration.

Features:
  - Schema validation with detailed error reporting
  - Migration generation and management
  - TypeScript SDK generation
  - LSP server for editor integration
  - Push/pull schemas to and from the CMS server
  - Seed content documents from JSON or YAML files
  - Manage workspace API keys`,
	Version: Version,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .fsl.yaml)")
	rootCmd.SetVersionTemplate(fmt.Sprintf("fsl version %s (built %s)\n", Version, BuildDate))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		loadConfig(cfgFile)
	} else {
		// Search for config file in current directory and parents
		cwd, err := os.Getwd()
		if err != nil {
			return
		}

		configPath := findConfigFile(cwd)
		if configPath != "" {
			loadConfig(configPath)
		}
	}
}

func findConfigFile(dir string) string {
	for {
		configPath := filepath.Join(dir, ".fsl.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		configPath = filepath.Join(dir, ".fsl.yml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		configPath = filepath.Join(dir, ".fluxcms.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		// Also check for .fluxcms.yml
		configPath = filepath.Join(dir, ".fluxcms.yml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func loadConfig(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	config = &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse config file: %v\n", err)
		config = nil
	}
}

// GetConfig returns the loaded configuration
func GetConfig() *Config {
	return config
}

// GetSchemaDirectory returns the configured schema directory or default
func GetSchemaDirectory() string {
	if config != nil && config.Schemas.Directory != "" {
		return config.Schemas.Directory
	}
	return "./schemas"
}
