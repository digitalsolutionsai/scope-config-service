package main

import (
	"context"
	"fmt"
	"log"

	"github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var getCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

		// Set up a connection to the server.
		conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		// Contact the server and print out its response.
		r, err := c.GetConfig(context.Background(), &configv1.GetConfigRequest{
			Identifier: &configv1.ConfigIdentifier{
				ServiceName: serviceName,
				ProjectId:   projectID,
				StoreId:     storeID,
				GroupId:     groupID,
				Scope:       configv1.Scope(configv1.Scope_value[scope]),
			},
		})

		if err != nil {
			log.Fatalf("could not get config: %v", err)
		}

		for _, field := range r.Fields {
			if field.Path == key {
				fmt.Println(field.Value)
				return
			}
		}

		fmt.Println("key not found")
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
