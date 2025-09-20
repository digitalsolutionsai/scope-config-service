package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"

	"github.com/spf13/cobra"
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish [version]",
	Short: "Publish a specific version of a configuration",
	Long:  `Marks a specific version of a configuration as 'published', making it the default version for clients.`,
	Example: `  # Publish version 2 of a configuration for a project
  config-cli publish 2 --service-name=api --scope=PROJECT --project-id=proj_123

  # Publish a version with a user name for auditing
  config-cli publish 3 --user-name="Jane Smith" --service-name=backend --scope=SYSTEM`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		identifier, err := createIdentifier()
		if err != nil {
			log.Fatalf("Error creating identifier: %v", err)
		}

		versionToPublish, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			log.Fatalf("Invalid version number: %v", err)
		}

		conn, err := getGrpcConn()
		if err != nil {
			log.Fatalf("Error connecting to gRPC server: %v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		req := &configv1.PublishVersionRequest{
			Identifier:       identifier,
			VersionToPublish: int32(versionToPublish),
			User:             userName,
		}

		resp, err := c.PublishVersion(context.Background(), req)
		if err != nil {
			log.Fatalf("could not publish version: %v", err)
		}

		fmt.Printf("Successfully published version %d for service %s\n", resp.PublishedVersion, resp.Identifier.ServiceName)
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)
}
