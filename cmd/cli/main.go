package main

import (
	"fmt"
	"os"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const cliVersion = "0.1.0"

var (
	cfgFile     string
	projectID   string
	serviceName string
	storeID     string
	userID      string
	scope       string
	userName    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "config-cli",
	Version: cliVersion,
	Short:   "A CLI for interacting with the Scope Config Service",
	Long: `A command-line interface for managing configurations in the Scope Config Service.

This CLI allows you to manage the entire lifecycle of a configuration, from creation and updates to publishing and viewing history.

Global Flags:
  --service-name: (Required) The name of the service to which the configuration belongs.
  --scope:        (Required) The configuration scope (e.g., SYSTEM, PROJECT, STORE, USER).
  --project-id:   The ID for the PROJECT scope.
  --store-id:     The ID for the STORE scope.
  --user-id:      The ID for the USER scope.
  --user-name:    The name of the user performing the action (for audit trails).

Example:
  config-cli set --service-name=api --scope=PROJECT --project-id=proj_123 db.user=admin
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if serviceName == "" {
			return fmt.Errorf("--service-name is a required flag")
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config-cli.yaml)")
	rootCmd.PersistentFlags().StringVar(&projectID, "project-id", "", "ID for the PROJECT scope")
	rootCmd.PersistentFlags().StringVar(&serviceName, "service-name", "", "Service name (required)")
	rootCmd.PersistentFlags().StringVar(&storeID, "store-id", "", "ID for the STORE scope")
	rootCmd.PersistentFlags().StringVar(&userID, "user-id", "", "ID for the USER scope")
	rootCmd.PersistentFlags().StringVar(&scope, "scope", "", "Configuration scope (SYSTEM, PROJECT, STORE, USER)")
	rootCmd.PersistentFlags().StringVar(&userName, "user-name", "", "User name for audit trails")

	rootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigName(".config-cli")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func createIdentifier() (*configv1.ConfigIdentifier, error) {
	if scope == "" {
		return nil, fmt.Errorf("--scope is a required flag")
	}

	scopeEnum, ok := configv1.Scope_value[scope]
	if !ok {
		return nil, fmt.Errorf("invalid scope: %s. valid scopes are: SYSTEM, PROJECT, STORE, USER", scope)
	}

	identifier := &configv1.ConfigIdentifier{
		ServiceName: serviceName,
		Scope:       configv1.Scope(scopeEnum),
	}

	// Associate the correct ID with the scope
	switch configv1.Scope(scopeEnum) {
	case configv1.Scope_PROJECT:
		if projectID == "" {
			return nil, fmt.Errorf("--project-id must be set for PROJECT scope")
		}
		identifier.ProjectId = projectID
	case configv1.Scope_STORE:
		if storeID == "" {
			return nil, fmt.Errorf("--store-id must be set for STORE scope")
		}
		identifier.StoreId = storeID
	case configv1.Scope_USER:
		if userID == "" {
			return nil, fmt.Errorf("--user-id must be set for USER scope")
		}
		identifier.UserId = userID
	}

	return identifier, nil
}
