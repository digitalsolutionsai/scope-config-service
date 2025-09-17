package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	projectID   string
	serviceName string
	storeID     string
	groupID     string
	scope       string
	userName    string // Added userName variable
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "config",
	Short: "A CLI for interacting with the Scope Config Service",
	Long:  `A command-line interface for managing configurations in the Scope Config Service.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config.yaml)")
	rootCmd.PersistentFlags().StringVar(&projectID, "project", "", "Project ID")
	rootCmd.PersistentFlags().StringVar(&serviceName, "service", "", "Service name")
	rootCmd.PersistentFlags().StringVar(&storeID, "store", "", "Store ID")
	rootCmd.PersistentFlags().StringVar(&groupID, "group", "", "Group ID")
	rootCmd.PersistentFlags().StringVar(&scope, "scope", "", "Scope (default, system, service, project, store)")
	rootCmd.PersistentFlags().StringVar(&userName, "user", "", "User name for audit trails") // Added user flag

	viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
	viper.BindPFlag("service", rootCmd.PersistentFlags().Lookup("service"))
	viper.BindPFlag("store", rootCmd.PersistentFlags().Lookup("store"))
	viper.BindPFlag("group", rootCmd.PersistentFlags().Lookup("group"))
	viper.BindPFlag("scope", rootCmd.PersistentFlags().Lookup("scope"))
	viper.BindPFlag("user", rootCmd.PersistentFlags().Lookup("user"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".config" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
