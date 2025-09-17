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

var setCmd = &cobra.Command{
	Use:   "set [key-value pairs]...",
	Short: "Set one or more configuration values",
	Example: `config set --service=my-app --scope=SYSTEM db.host=localhost db.port=5432`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("at least one key-value pair must be provided")
		}

		var fields []*configv1.ConfigField
		for _, arg := range args {
			parts := splitKeyValue(arg)
			if len(parts) != 2 {
				log.Fatalf("invalid key-value pair: %s", arg)
			}
			fields = append(fields, &configv1.ConfigField{Path: parts[0], Value: parts[1]})
		}


		conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		req := &configv1.UpdateConfigRequest{
			Identifier: &configv1.ConfigIdentifier{
				ServiceName: serviceName,
				ProjectId:   projectID,
				StoreId:     storeID,
				GroupId:     groupID,
				Scope:       configv1.Scope(configv1.Scope_value[scope]),
			},
			Fields: fields,
			User:   userName,
		}

		resp, err := c.UpdateConfig(context.Background(), req)
		if err != nil {
			log.Fatalf("could not set config: %v", err)
		}

		fmt.Printf("Successfully updated config for %s. New version: %d\n", resp.VersionInfo.Identifier.ServiceName, resp.CurrentVersion)
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
}

// splitKeyValue splits a string by the first equals sign.
func splitKeyValue(s string) []string {
	for i, r := range s {
		if r == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
