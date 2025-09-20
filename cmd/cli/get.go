package main

import (
	"context"
	"fmt"
	"log"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"

	"github.com/spf13/cobra"
)

var (
	configVersion int32
	latest        bool
	path          string
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a configuration, optionally for a specific version or the latest",
	Example: `  # Get the published configuration for a project
  config-cli get --service-name=my-service --scope=PROJECT --project-id=proj_123

  # Get a single key from the published configuration
  config-cli get --service-name=my-service --scope=PROJECT --project-id=proj_123 --path=my.key

  # Get the latest (active) configuration for a user
  config-cli get --latest --service-name=my-service --scope=USER --user-id=user_456

  # Get a specific version of a configuration for a store
  config-cli get --version=3 --service-name=my-service --scope=STORE --store-id=store_789`,
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := getGrpcConn()
		if err != nil {
			log.Fatalf("Error connecting to gRPC server: %v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		identifier, err := createIdentifier()
		if err != nil {
			log.Fatalf("Error creating identifier: %v", err)
		}

		var resp *configv1.ScopeConfig

		switch {
		case latest:
			resp, err = c.GetLatestConfig(context.Background(), &configv1.GetConfigRequest{Identifier: identifier, Path: path})
		case configVersion > 0:
			resp, err = c.GetConfigByVersion(context.Background(), &configv1.GetConfigByVersionRequest{Identifier: identifier, Version: configVersion, Path: path})
		default:
			resp, err = c.GetConfig(context.Background(), &configv1.GetConfigRequest{Identifier: identifier, Path: path})
		}

		if err != nil {
			log.Fatalf("could not get config: %v", err)
		}

		if path != "" {
			if len(resp.Fields) > 0 {
				fmt.Println(resp.Fields[0].Value)
			} else {
				fmt.Println("null")
			}
			return
		}

		fmt.Printf("Version Info: %v\n", resp.VersionInfo)
		fmt.Printf("Current Version: %d\n", resp.CurrentVersion)
		for _, field := range resp.Fields {
			fmt.Printf("  %s: %s\n", field.Path, field.Value)
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().BoolVar(&latest, "latest", false, "Get the latest version of the configuration")
	getCmd.Flags().Int32Var(&configVersion, "version", 0, "Get a specific version of the configuration")
	getCmd.Flags().StringVar(&path, "path", "", "Get a single key from the configuration")
	getCmd.MarkFlagsMutuallyExclusive("latest", "version")
}
