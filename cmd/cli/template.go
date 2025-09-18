package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

var templateFilePath string

// yamlField is a temporary struct for unmarshaling the YAML file.
// It uses strings for fields that are enums in the proto definition.
type yamlField struct {
	Path         string   `yaml:"path"`
	Label        string   `yaml:"label"`
	Description  string   `yaml:"description"`
	Type         string   `yaml:"type"`
	DefaultValue string   `yaml:"defaultValue"`
	DisplayOn    []string `yaml:"displayOn"`
}

// yamlTemplate is a temporary struct for unmarshaling the YAML file.
type yamlTemplate struct {
	Identifier *configv1.ConfigIdentifier `yaml:"identifier"`
	Fields     []yamlField                `yaml:"fields"`
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage configuration templates",
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a configuration template from a file",
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
		if err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()
		client := configv1.NewConfigServiceClient(conn)

		yamlFile, err := ioutil.ReadFile(templateFilePath)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}

		var temp yamlTemplate
		err = yaml.Unmarshal(yamlFile, &temp)
		if err != nil {
			log.Fatalf("Failed to unmarshal YAML: %v", err)
		}

		// Manually construct the final ConfigTemplate from the temporary struct.
		template := &configv1.ConfigTemplate{
			Identifier: temp.Identifier,
			Fields:     make([]*configv1.ConfigFieldTemplate, len(temp.Fields)),
		}

		for i, yf := range temp.Fields {
			template.Fields[i] = &configv1.ConfigFieldTemplate{
				Path:         yf.Path,
				Label:        yf.Label,
				Description:  yf.Description,
				DefaultValue: yf.DefaultValue,
				Type:         configv1.FieldType(configv1.FieldType_value[yf.Type]),
				DisplayOn:    make([]configv1.Scope, len(yf.DisplayOn)),
			}
			for j, s := range yf.DisplayOn {
				template.Fields[i].DisplayOn[j] = configv1.Scope(configv1.Scope_value[s])
			}
		}

		req := &configv1.ApplyConfigTemplateRequest{
			Template: template,
			User:     userName,
		}

		resp, err := client.ApplyConfigTemplate(context.Background(), req)
		if err != nil {
			log.Fatalf("Error calling ApplyConfigTemplate: %v", err)
		}

		fmt.Printf("Successfully applied template for service %s and group %s\n", resp.Identifier.ServiceName, resp.Identifier.GroupId)
	},
}

func init() {
	applyCmd.Flags().StringVarP(&templateFilePath, "file", "f", "", "Path to the template YAML file")
	applyCmd.MarkFlagRequired("file")
	templateCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(templateCmd)
}
