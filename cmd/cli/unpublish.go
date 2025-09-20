package main

import (
	"context"
	"fmt"
	"log"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"

	"github.com/spf13/cobra"
)

// unpublishCmd represents the unpublish command
var unpublishCmd = &cobra.Command{
	Use:   "unpublish",
	Short: "Unpublishes the configuration for a given scope",
	Long:  `Marks the configuration for a given scope as unpublished. This effectively removes the default configuration, but preserves the history.`,
	Example: `  # Unpublish the configuration for a project
  config-cli unpublish --service-name=api --scope=PROJECT --project-id=proj_123`,
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

		req := &configv1.PublishVersionRequest{
			Identifier:       identifier,
			VersionToPublish: 0, // Publishing version 0 is interpreted as unpublishing
			User:             userName,
		}

		_, err = c.PublishVersion(context.Background(), req)
		if err != nil {
			log.Fatalf("could not unpublish configuration: %v", err)
		}

		fmt.Printf("Successfully unpublished configuration for service %s\n", serviceName)
	},
}

func init() {
	rootCmd.AddCommand(unpublishCmd)
}
