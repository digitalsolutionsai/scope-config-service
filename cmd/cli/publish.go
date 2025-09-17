package main

import (
	"context"
	"fmt"
	"log"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var publishCmd = &cobra.Command{
	Use:   "publish [version]",
	Short: "Publish a specific version of a configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version := args[0]

		conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		// Convert the version string to an int32
		var versionToPublish int32
		fmt.Sscanf(version, "%d", &versionToPublish)

		req := &configv1.PublishVersionRequest{
			Identifier: &configv1.ConfigIdentifier{
				ServiceName: serviceName,
				ProjectId:   projectID,
				StoreId:     storeID,
				GroupId:     groupID,
				Scope:       configv1.Scope(configv1.Scope_value[scope]),
			},
			VersionToPublish: versionToPublish,
			User:             userName,
		}

		resp, err := c.PublishVersion(context.Background(), req)
		if err != nil {
			log.Fatalf("could not publish config: %v", err)
		}

		fmt.Printf("Successfully published version %d for %s\n", resp.PublishedVersion, resp.Identifier.ServiceName)
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)
}
