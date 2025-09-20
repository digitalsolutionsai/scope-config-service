package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"

	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set [key=value]...",
	Short: "Set one or more configuration values",
	Long:  `Set one or more configuration values for a given scope.`,
	Example: `  # Set a single value for a project and group
  config-cli set --service-name=api --scope=PROJECT --project-id=proj_123 --group-id=db --user-name="John Doe" db.user=admin

  # Set multiple values for a store and group
  config-cli set --service-name=webapp --scope=STORE --store-id=store_789 --group-id=stripe --user-name="Jane Doe" stripe.key=pk_... stripe.secret=sk_...

  # Set values for the system scope
  config-cli set --service-name=backend --scope=SYSTEM --user-name="Admin" api.key=...`,
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

		conn, err := getGrpcConn()
		if err != nil {
			log.Fatalf("Error connecting to gRPC server: %v", err)
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
