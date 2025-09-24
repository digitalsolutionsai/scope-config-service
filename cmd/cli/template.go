package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var templateFilePath string

// Intermediate structs for YAML parsing
type YamlValueOption struct {
	Value string `yaml:"value"`
	Label string `yaml:"label"`
}

type YamlFieldTemplate struct {
	Path         string            `yaml:"path"`
	Label        string            `yaml:"label"`
	Description  string            `yaml:"description"`
	Type         string            `yaml:"type"`
	DefaultValue string            `yaml:"defaultValue"`
	DisplayOn    []string          `yaml:"displayOn"`
	Options      []YamlValueOption `yaml:"options"`
}

type YamlGroup struct {
	ID          string              `yaml:"id"`
	Label       string              `yaml:"label"`
	Description string              `yaml:"description"`
	Fields      []YamlFieldTemplate `yaml:"fields"`
}

type YamlTemplate struct {
	Service struct {
		Name  string `yaml:"id"`
		Label string `yaml:"label"`
	} `yaml:"service"`
	Groups []YamlGroup `yaml:"groups"`
}

// Mapping functions
func toFieldType(s string) configv1.FieldType {
	if val, ok := configv1.FieldType_value[s]; ok {
		return configv1.FieldType(val)
	}
	return configv1.FieldType_FIELD_TYPE_UNSPECIFIED
}

func toScope(s string) configv1.Scope {
	if val, ok := configv1.Scope_value[s]; ok {
		return configv1.Scope(val)
	}
	return configv1.Scope_SCOPE_UNSPECIFIED
}

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
	Long:    `Applies a configuration template from a YAML file.`,
	Example: `  # Apply a template from a file
  config-cli template apply -f ./templates/example_payment.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		yamlFile, err := ioutil.ReadFile(templateFilePath)
		if err != nil {
			log.Fatalf("Error reading template file: %v", err)
		}

		var yamlTemplate YamlTemplate
		if err := yaml.Unmarshal(yamlFile, &yamlTemplate); err != nil {
			log.Fatalf("Error unmarshalling YAML: %v", err)
		}

		conn, err := getGrpcConn()
		if err != nil {
			log.Fatalf("%v", err)
		}
		defer conn.Close()
		c := configv1.NewConfigServiceClient(conn)

		for _, group := range yamlTemplate.Groups {
			fields := make([]*configv1.ConfigFieldTemplate, len(group.Fields))
			for i, f := range group.Fields {
				displayOn := make([]configv1.Scope, len(f.DisplayOn))
				for j, d := range f.DisplayOn {
					displayOn[j] = toScope(d)
				}

				options := make([]*configv1.ValueOption, len(f.Options))
				for j, o := range f.Options {
					options[j] = &configv1.ValueOption{Value: o.Value, Label: o.Label}
				}

				fields[i] = &configv1.ConfigFieldTemplate{
					Path:         f.Path,
					Label:        f.Label,
					Description:  f.Description,
					Type:         toFieldType(f.Type),
					DefaultValue: f.DefaultValue,
					DisplayOn:    displayOn,
					Options:      options,
				}
			}

			template := &configv1.ConfigTemplate{
				Identifier: &configv1.ConfigIdentifier{
					ServiceName: yamlTemplate.Service.Name,
					GroupId:     group.ID,
				},
				ServiceLabel:     yamlTemplate.Service.Label,
				GroupLabel:       group.Label,
				GroupDescription: group.Description,
				Fields:           fields,
			}

			req := &configv1.ApplyConfigTemplateRequest{
				Template: template,
				User:     userName,
			}

			_, err = c.ApplyConfigTemplate(context.Background(), req)
			if err != nil {
				log.Fatalf("could not apply template for group %s: %v", group.ID, err)
			}
			fmt.Printf("Successfully applied configuration template for group '%s' in service '%s'.\n", group.ID, yamlTemplate.Service.Name)
		}
	},
}

func init() {
	applyCmd.Flags().StringVarP(&templateFilePath, "file", "f", "", "Path to the template file (required)")
	applyCmd.MarkFlagRequired("file")
	templateCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(templateCmd)
}
