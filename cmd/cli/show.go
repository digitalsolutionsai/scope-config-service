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
	history bool
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the configuration history or the active/published versions",
	Long:  `Displays the version history or compares the active (latest) and published configurations for a given scope.`,
	Example: `  # Show the active and published versions for a project
  config-cli show --service-name=my-app --scope=PROJECT --project-id=proj-abc

  # Show the full version history for a user's configuration
  config-cli show --history --service-name=my-app --scope=USER --user-id=user-123`,
	Run: func(cmd *cobra.Command, args []string) {
		identifier, err := createIdentifier()
		if err != nil {
			log.Fatalf("Error creating identifier: %v", err)
		}

		conn, err := getGrpcConn()
		if err != nil {
			log.Fatalf("Error connecting to gRPC server: %v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		if history {
			showHistory(c, identifier)
		} else {
			showActiveAndPublished(c, identifier)
		}
	},
}

func showHistory(client configv1.ConfigServiceClient, identifier *configv1.ConfigIdentifier) {
	// Get the published config to identify the published version in the history
	publishedResp, err := client.GetConfig(context.Background(), &configv1.GetConfigRequest{Identifier: identifier})
	var publishedVersion int32 = -1 // Use an impossible version number if no published version exists
	if err == nil && publishedResp != nil {
		publishedVersion = publishedResp.GetCurrentVersion()
	} else {
		log.Println("Could not retrieve published configuration to mark it in history.")
	}

	req := &configv1.GetConfigHistoryRequest{Identifier: identifier}
	resp, err := client.GetConfigHistory(context.Background(), req)
	if err != nil {
		log.Fatalf("could not get config history: %v", err)
	}

	if len(resp.Versions) == 0 {
		fmt.Println("No version history found.")
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '	', 0)
	fmt.Fprintln(w, "Version\tStatus\tUpdated At\tUpdated By")
	for _, v := range resp.Versions {
		status := ""
		if v.GetLatestVersion() == publishedVersion {
			status = "Published"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", v.GetLatestVersion(), status, formatTimestamp(v.GetUpdatedAt()), v.GetUpdatedBy())
	}
	w.Flush()
}

func showActiveAndPublished(client configv1.ConfigServiceClient, identifier *configv1.ConfigIdentifier) {
	// Get the active (latest) config
	activeResp, err := client.GetLatestConfig(context.Background(), &configv1.GetConfigRequest{Identifier: identifier})
	if err != nil {
		log.Fatalf("could not get active config: %v", err)
	}

	// Get the published config
	publishedResp, err := client.GetConfig(context.Background(), &configv1.GetConfigRequest{Identifier: identifier})
	if err != nil {
		log.Fatalf("could not get published config: %v", err)
	}

	fmt.Println("--- Active Configuration (Latest) ---")
	printConfigResponse(activeResp)

	fmt.Println("\n--- Published Configuration ---")
	printConfigResponse(publishedResp)
}

func printConfigResponse(resp *configv1.ScopeConfig) {
	if resp == nil {
		fmt.Println("No configuration found.")
		return
	}
	fmt.Printf("Version: %d\n", resp.GetCurrentVersion())
	fmt.Printf("Updated At: %s\n", formatTimestamp(resp.GetVersionInfo().GetUpdatedAt()))
	fmt.Printf("Updated By: %s\n", resp.GetVersionInfo().GetUpdatedBy())
	fmt.Println("Fields:")
	for _, field := range resp.GetFields() {
		fmt.Printf("  %s: %s\n", field.GetPath(), field.GetValue())
	}
}

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "N/A"
	}
	return ts.AsTime().Format(time.RFC3339)
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&history, "history", false, "Show the entire version history")
}
