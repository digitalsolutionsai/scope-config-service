package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set [key=value]...",
	Short: "Set one or more configuration values",
	Long:  `Set one or more configuration values for a given scope.`,
	Example: `  # Set a single value for a project
  config-cli set --service-name=api --scope=PROJECT --project-id=proj_123 db.user=admin

  # Set multiple values for a store
  config-cli set --service-name=webapp --scope=STORE --store-id=store_789 stripe.key=pk_... stripe.secret=sk_...

  # Set values with a user name for auditing
  config-cli set --user-name="John Doe" --service-name=backend --scope=SYSTEM api.key=...`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		identifier, err := createIdentifier()
		if err != nil {
			log.Fatalf("Error creating identifier: %v", err)
		}

		var fields []*configv1.ConfigField
		for _, arg := range args {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				log.Fatalf("invalid key-value pair: %s. Must be in 'key=value' format", arg)
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
			Identifier: identifier,
			Fields:     fields,
			User:       userName,
		}

		resp, err := c.UpdateConfig(context.Background(), req)
		if err != nil {
			log.Fatalf("could not update config: %v", err)
		}

		fmt.Printf("Successfully updated config. New version: %d\n", resp.CurrentVersion)
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
}
