package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	Example: `  # Get the published configuration for a project and group
  config-cli get --service-name=my-service --scope=PROJECT --project-id=proj_123 --group-id=features

  # Get a single key from the published configuration
  config-cli get --service-name=my-service --scope=PROJECT --project-id=proj_123 --group-id=features --path=my.key

  # Get the latest configuration for a user and group
  config-cli get --latest --service-name=my-service --scope=USER --user-id=user_456 --group-id=settings

  # Get a specific version of a configuration for a store and group
  config-cli get --version=3 --service-name=my-service --scope=STORE --store-id=store_789 --group-id=checkout`,
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

		printGetResponse(resp)
	},
}

func printGetResponse(resp *configv1.ScopeConfig) {
	if resp == nil || resp.GetCurrentVersion() == 0 {
		fmt.Println("No configuration found.")
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintf(w, "Version:\t%d\n", resp.GetCurrentVersion())
	status := "Unpublished"
	if resp.GetVersionInfo().GetPublishedVersion() == resp.GetCurrentVersion() {
		status = "Published"
	}
	fmt.Fprintf(w, "Status:\t%s\n", status)
	fmt.Fprintf(w, "Updated At:\t%s\n", formatTimestamp(resp.GetVersionInfo().GetUpdatedAt()))
	fmt.Fprintf(w, "Updated By:\t%s\n", resp.GetVersionInfo().GetUpdatedBy())
	w.Flush()

	if len(resp.GetFields()) > 0 {
		fmt.Println("\nFields:")
		for _, field := range resp.GetFields() {
			fmt.Printf("  %s: %s\n", field.GetPath(), field.GetValue())
		}
	}
}

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "N/A"
	}
	return ts.AsTime().Format(time.RFC3339)
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().BoolVar(&latest, "latest", false, "Get the latest version of the configuration")
	getCmd.Flags().Int32Var(&configVersion, "version", 0, "Get a specific version of the configuration")
	getCmd.Flags().StringVar(&path, "path", "", "Get a single key from the configuration")
	getCmd.MarkFlagsMutuallyExclusive("latest", "version")
}
