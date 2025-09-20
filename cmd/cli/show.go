package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
)

var (
	history bool
	limit   int
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the configuration history or the latest/published versions",
	Long:  `Displays the version history or compares the latest and published configurations for a given scope.`,
	Example: `  # Show the latest and published versions for a project and group
  config-cli show --service-name=my-app --scope=PROJECT --project-id=proj-abc --group-id=features

  # Show the full version history for a user's configuration
  config-cli show --history --service-name=my-app --scope=USER --user-id=user-123 --group-id=settings

  # Show the last 5 versions of the history
  config-cli show --history --limit=5 --service-name=my-app --scope=USER --user-id=user-123 --group-id=settings`,
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
			showLatestAndPublished(c, identifier)
		}
	},
}

func showHistory(client configv1.ConfigServiceClient, identifier *configv1.ConfigIdentifier) {
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
	w.Init(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "Version\tStatus\tUpdated At\tUpdated By")

	versionsToShow := resp.Versions
	if limit > 0 && len(versionsToShow) > limit {
		versionsToShow = versionsToShow[:limit]
	}

	for _, v := range versionsToShow {
		status := "Unpublished"
		if v.GetPublishedVersion() == v.GetLatestVersion() {
			status = "Published"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", v.GetLatestVersion(), status, formatTimestamp(v.GetUpdatedAt()), v.GetUpdatedBy())
	}
	w.Flush()
}

func showLatestAndPublished(client configv1.ConfigServiceClient, identifier *configv1.ConfigIdentifier) {
	// Get the latest config
	latestResp, err := client.GetLatestConfig(context.Background(), &configv1.GetConfigRequest{Identifier: identifier})
	if err != nil {
		log.Fatalf("could not get latest config: %v", err)
	}

	// Get the published config
	publishedResp, err := client.GetConfig(context.Background(), &configv1.GetConfigRequest{Identifier: identifier})
	if err != nil {
		log.Fatalf("could not get published config: %v", err)
	}

	fmt.Println("--- Latest Configuration ---")
	printGetResponse(latestResp)

	// Avoid printing the same details twice if latest is also published
	if latestResp.GetCurrentVersion() != publishedResp.GetCurrentVersion() {
		fmt.Println("\n--- Published Configuration ---")
		printGetResponse(publishedResp)
	}
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&history, "history", false, "Show the entire version history")
	showCmd.Flags().IntVar(&limit, "limit", 100, "Limit the number of history versions to show")
}
