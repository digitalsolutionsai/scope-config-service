package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
)

var limit int32

// historyCmd represents the history command
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Display the version history of a configuration",
	Long:  `Fetches and displays the version history for a given configuration identifier, showing who made changes and when.`,
	Run: func(cmd *cobra.Command, args []string) {
		identifier, err := createIdentifier()
		if err != nil {
			log.Fatalf("Error creating identifier: %v", err)
		}

		conn, err := getGrpcConn()
		if err != nil {
			log.Fatalf("%v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		req := &configv1.GetConfigHistoryRequest{Identifier: identifier, Limit: limit}

		resp, err := c.GetConfigHistory(context.Background(), req)
		if err != nil {
			log.Fatalf("could not get config history: %v", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "VERSION\tCREATED AT\tCREATED BY")

		for _, entry := range resp.History {
			createdAt := entry.CreatedAt.AsTime().Format("2006-01-02 15:04:05")
			fmt.Fprintf(w, "%d\t%s\t%s\n", entry.Version, createdAt, entry.CreatedBy)
		}

		w.Flush()
	},
}

func init() {
	historyCmd.Flags().Int32Var(&limit, "limit", 10, "Limit the number of history entries to return")
	rootCmd.AddCommand(historyCmd)
}
