package main

import (
	"context"
	"fmt"
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
	"log"
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the latest and published configurations",
	Long:  `Displays and compares the latest and published configurations for a given scope.`,
	Example: `  # Show the latest and published versions for a project and group
  config-cli show --service-name=my-app --scope=PROJECT --project-id=proj-abc --group-id=features`,
	Run: func(cmd *cobra.Command, args []string) {
		identifier, err := createIdentifier()
		if err != nil {
			log.Fatalf("Error creating identifier: %v", err)
		}

		conn, err := getGrpcConn()
		if err != nil {
			log.Fatalf("%v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		showLatestAndPublished(c, identifier)
	},
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
}
