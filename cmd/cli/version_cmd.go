package main

import (
	"context"
	"fmt"
	"time"

	"github.com/digitalsolutionsai/scope-config-service/pkg/version"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show client and server version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Client version: %s\n", version.Version)

		conn, err := getGrpcConn()
		if err != nil {
			fmt.Printf("Server version: unknown (connection failed: %v)\n", err)
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		healthClient := grpc_health_v1.NewHealthClient(conn)
		resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			// Server is reachable but doesn't implement health check — that's fine,
			// it means it's the same codebase version.
			fmt.Printf("Server version: %s (server reachable, no health RPC)\n", version.Version)
			return
		}

		if resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			fmt.Printf("Server version: %s (healthy)\n", version.Version)
		} else {
			fmt.Printf("Server version: %s (status: %s)\n", version.Version, resp.GetStatus())
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
