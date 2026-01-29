package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xtruder/sy/internal/api"
	"github.com/xtruder/sy/internal/config"
)

var (
	cfgFile         string
	client          *api.Client
	currentNotebook *api.Notebook
	configDir       string

	// Version info (set via SetVersion)
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// SetVersion sets the version information (called from main)
func SetVersion(v, c, d string) {
	version = v
	commit = c
	date = d
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(`{{.Name}} {{.Version}}
Commit: ` + commit + `
Built:  ` + date + "\n")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sy",
	Short: "SiYuan CLI - Command-line interface for SiYuan Note",
	Long: `sy is a command-line tool for interacting with SiYuan Note.

It provides commands for listing, exporting, importing, and editing
documents, as well as running SQL queries and full-text searches.

Designed for power users and LLM integration.

Notebook Selection (in order of priority):
  1. --notebook flag
  2. SIYUAN_NOTEBOOK environment variable
  3. notebook setting in config file
  4. First available notebook`,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip client initialization for help and completion commands
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}

		// Initialize API client
		host := viper.GetString("host")
		token := viper.GetString("token")

		var err error
		client, err = api.NewClient(host, token)
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		// Resolve current notebook (skip for 'notebook' command which handles this itself)
		if cmd.Name() != "notebook" && cmd.Parent() != nil && cmd.Parent().Name() != "notebook" {
			if err := resolveCurrentNotebook(); err != nil {
				// Don't fail for commands that might not need a notebook
				// Just log a warning
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/sy/config.yaml)")
	rootCmd.PersistentFlags().String("host", "", "SiYuan API host (default: http://localhost:6806)")
	rootCmd.PersistentFlags().String("token", "", "SiYuan API token")
	rootCmd.PersistentFlags().StringP("notebook", "n", "", "Notebook name or ID")

	// Bind flags to viper
	_ = viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("host"))
	_ = viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	_ = viper.BindPFlag("notebook", rootCmd.PersistentFlags().Lookup("notebook"))

	// Add subcommands
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(mvCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(sqlCmd)
	rootCmd.AddCommand(notebookCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not find home directory:", err)
		return
	}

	configDir = filepath.Join(home, ".config", "sy")

	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in ~/.config/sy/
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("SIYUAN")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("host", config.DefaultHost)

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is not an error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Warning: error reading config file:", err)
		}
	}
}

// GetClient returns the initialized API client
func GetClient() *api.Client {
	return client
}

// GetCurrentNotebook returns the current notebook
func GetCurrentNotebook() *api.Notebook {
	return currentNotebook
}

// GetConfigDir returns the config directory path
func GetConfigDir() string {
	return configDir
}

// resolveCurrentNotebook determines the current notebook based on flag/env/config/default
func resolveCurrentNotebook() error {
	notebookRef := viper.GetString("notebook")

	notebooks, err := client.ListNotebooks()
	if err != nil {
		return fmt.Errorf("failed to list notebooks: %w", err)
	}

	if len(notebooks) == 0 {
		return fmt.Errorf("no notebooks found")
	}

	// Filter out closed notebooks
	var openNotebooks []*api.Notebook
	for _, nb := range notebooks {
		if !nb.Closed {
			openNotebooks = append(openNotebooks, nb)
		}
	}

	if len(openNotebooks) == 0 {
		return fmt.Errorf("no open notebooks found")
	}

	if notebookRef == "" {
		// Use first open notebook as default
		currentNotebook = openNotebooks[0]
		return nil
	}

	// Try to find by name or ID
	for _, nb := range openNotebooks {
		if nb.Name == notebookRef || nb.ID == notebookRef {
			currentNotebook = nb
			return nil
		}
	}

	return fmt.Errorf("notebook not found: %s", notebookRef)
}

// SetCurrentNotebook sets the current notebook and optionally saves to config
func SetCurrentNotebook(notebook *api.Notebook, save bool) error {
	currentNotebook = notebook

	if save {
		return SaveNotebookToConfig(notebook.Name)
	}
	return nil
}

// SaveNotebookToConfig saves the notebook name to the config file
func SaveNotebookToConfig(notebookName string) error {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	viper.Set("notebook", notebookName)

	configPath := filepath.Join(configDir, "config.yaml")
	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
