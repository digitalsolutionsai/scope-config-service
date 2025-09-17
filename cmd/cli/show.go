package main

import (
	"context"
	"fmt"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the latest (unpublished) configuration",
	Long:  `Retrieves and displays the latest version of a configuration, regardless of its publication status.`,
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Printf("Failed to connect: %v\n", err)
			return
		}
		defer conn.Close()

		client := configv1.NewConfigServiceClient(conn)

		identifier := &configv1.ConfigIdentifier{
			ServiceName: serviceName,
			ProjectId:   projectID,
			StoreId:     storeID,
			GroupId:     groupID,
			Scope:       configv1.Scope(configv1.Scope_value[scope]),
		}

		req := &configv1.GetConfigRequest{Identifier: identifier}

		resp, err := client.GetLatestConfig(context.Background(), req)
		if err != nil {
			fmt.Printf("Error calling GetLatestConfig: %v\n", err)
			return
		}

		fmt.Printf("Latest Version: %d\n", resp.CurrentVersion)
		fmt.Printf("Published Version: %d\n", resp.VersionInfo.PublishedVersion)
		fmt.Println("Fields:")
		for _, field := range resp.Fields {
			fmt.Printf("  %s: %s\n", field.Path, field.Value)
		}
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
