package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"
)

var templateFilePath string

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage configuration templates",
	Long:  `Provides subcommands to manage configuration templates.`,
}

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:     "apply -f [file]",
	Short:   "Apply a configuration template",
	Long:    `Applies a configuration template from a YAML file to the specified scope.`,
	Example: `  # Apply a template to a project
  config-cli template apply -f ./template.yaml --service-name=my-app --scope=PROJECT --project-id=proj-123

  # Create a template file (e.g., template.yaml)
  # fields:
  #   - path: "db.host"
  #     value: "localhost"
  #   - path: "db.port"
  #     value: "5432"`,
	Run: func(cmd *cobra.Command, args []string) {
		identifier, err := createIdentifier()
		if err != nil {
			log.Fatalf("Error creating identifier: %v", err)
		}

		yamlFile, err := ioutil.ReadFile(templateFilePath)
		if err != nil {
			log.Fatalf("Error reading template file: %v", err)
		}

		var template configv1.ConfigTemplate
		if err := yaml.Unmarshal(yamlFile, &template); err != nil {
			log.Fatalf("Error unmarshalling YAML: %v", err)
		}
		template.Identifier = identifier

		conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		req := &configv1.ApplyConfigTemplateRequest{
			Template: &template,
			User:     userName,
		}

		_, err = c.ApplyConfigTemplate(context.Background(), req)
		if err != nil {
			log.Fatalf("could not apply template: %v", err)
		}

		fmt.Println("Successfully applied configuration template.")
	},
}

func init() {
	applyCmd.Flags().StringVarP(&templateFilePath, "file", "f", "", "Path to the template file (required)")
	applyCmd.MarkFlagRequired("file")
	templateCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(templateCmd)
}
