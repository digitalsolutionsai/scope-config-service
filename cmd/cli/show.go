
package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	history bool
)

// result is a container for the output of a goroutine fetching a config.
type result struct {
	config *configv1.ScopeConfig
	err    error
	label  string
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show active and published configurations",
	Long:  `Retrieves and displays the active (latest) and the currently published configurations. Use --history to see all versions.`,
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

		if history {
			runShowHistory(client, identifier)
		} else {
			runShowActiveAndPublished(client, identifier)
		}
	},
}

func runShowHistory(client configv1.ConfigServiceClient, identifier *configv1.ConfigIdentifier) {
	req := &configv1.GetConfigHistoryRequest{Identifier: identifier}
	resp, err := client.GetConfigHistory(context.Background(), req)
	if err != nil {
		fmt.Printf("Error calling GetConfigHistory: %v\n", err)
		return
	}

	if len(resp.Versions) == 0 {
		fmt.Println("No version history found for the specified configuration.")
		return
	}

	// We need to know the published version to mark it in the table.
	// We can get this from the first entry in the history.
	publishedVersion := resp.Versions[0].PublishedVersion

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "Version\tStatus\tCreated At\tCreated By")
	fmt.Fprintln(w, "-------\t------\t----------\t----------")

	for _, v := range resp.Versions {
		status := ""
		if v.Id == publishedVersion {
			status = "Published"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", v.Id, status, formatTimestamp(v.CreatedAt), v.CreatedBy)
	}
	w.Flush()
}

func runShowActiveAndPublished(client configv1.ConfigServiceClient, identifier *configv1.ConfigIdentifier) {
	// Use a channel to receive results from goroutines
	ch := make(chan result, 2)

	// Get Active (Latest) Config
	go func() {
		req := &configv1.GetConfigRequest{Identifier: identifier}
		resp, err := client.GetLatestConfig(context.Background(), req)
		ch <- result{config: resp, err: err, label: "Active (Latest)"}
	}()

	// Get Published Config
	go func() {
		req := &configv1.GetConfigRequest{Identifier: identifier}
		resp, err := client.GetConfig(context.Background(), req)
		ch <- result{config: resp, err: err, label: "Published"}
	}()

	// Process results
	results := make(map[string]result)
	for i := 0; i < 2; i++ {
		r := <-ch
		results[r.label] = r
	}

	displayConfig(results["Active (Latest)"])
	fmt.Println("---")
	displayConfig(results["Published"])
}

func displayConfig(r result) {
	fmt.Printf("--- %s ---\n", r.label)
	if r.err != nil {
		fmt.Printf("Error: %v\n", r.err)
		return
	}
	if r.config == nil {
		fmt.Println("No configuration found.")
		return
	}

	fmt.Printf("Version: %d\n", r.config.CurrentVersion)
	fmt.Printf("Published Version: %d\n", r.config.VersionInfo.PublishedVersion)
	fmt.Println("Fields:")
	for _, field := range r.config.Fields {
		fmt.Printf("  %s: %s\n", field.Path, field.Value)
	}
}

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "N/A"
	}
	return ts.AsTime().Format(time.RFC3339)
}


func init() {
	showCmd.Flags().BoolVar(&history, "history", false, "Show the entire version history")
	rootCmd.AddCommand(showCmd)
}
