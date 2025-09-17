package main

import (
	"context"
	"log"

	"github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var setCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		// Set up a connection to the server.
		conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		// Contact the server and print out its response.
		_, err = c.UpdateConfig(context.Background(), &configv1.UpdateConfigRequest{
			Identifier: &configv1.ConfigIdentifier{
				ServiceName: serviceName,
				ProjectId:   projectID,
				StoreId:     storeID,
				GroupId:     groupID,
				Scope:       configv1.Scope(configv1.Scope_value[scope]),
			},
			Fields: []*configv1.ConfigField{
				{
					Path:  key,
					Value: value,
				},
			},
		})

		if err != nil {
			log.Fatalf("could not set config: %v", err)
		}

		log.Printf("key '%s' set successfully", key)
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
}
