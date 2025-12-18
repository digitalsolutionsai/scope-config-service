package scopeconfig

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
	"gopkg.in/yaml.v2"
)

// YamlValueOption represents a value option in YAML format.
type YamlValueOption struct {
	Value string `yaml:"value"`
	Label string `yaml:"label"`
}

// YamlFieldTemplate represents a field template in YAML format.
type YamlFieldTemplate struct {
	Path         string            `yaml:"path"`
	Label        string            `yaml:"label"`
	Description  string            `yaml:"description"`
	Type         string            `yaml:"type"`
	DefaultValue string            `yaml:"defaultValue"`
	DisplayOn    []string          `yaml:"displayOn"`
	Options      []YamlValueOption `yaml:"options"`
	SortOrder    int32             `yaml:"sortOrder"`
}

// YamlGroup represents a configuration group in YAML format.
type YamlGroup struct {
	ID          string              `yaml:"id"`
	Label       string              `yaml:"label"`
	Description string              `yaml:"description"`
	Fields      []YamlFieldTemplate `yaml:"fields"`
	SortOrder   int32               `yaml:"sortOrder"`
}

// YamlTemplate represents the complete template structure in YAML format.
type YamlTemplate struct {
	Service struct {
		Name  string `yaml:"id"`
		Label string `yaml:"label"`
	} `yaml:"service"`
	Groups []YamlGroup `yaml:"groups"`
}

// LoadTemplatesFromDir loads all YAML templates from the specified directory
// and applies them using the client.
//
// Example:
//
//	err := client.LoadTemplatesFromDir(ctx, "./templates", "system")
func (c *Client) LoadTemplatesFromDir(ctx context.Context, dir string, user string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("Templates directory %s does not exist, skipping template import", dir)
		return nil
	}

	var templateFiles []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yml" || ext == ".yaml" {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk templates directory: %w", err)
	}

	if len(templateFiles) == 0 {
		log.Printf("No template files found in %s", dir)
		return nil
	}

	log.Printf("Found %d template file(s) to import", len(templateFiles))

	for _, file := range templateFiles {
		if err := c.loadAndApplyFile(ctx, file, user); err != nil {
			return fmt.Errorf("failed to apply template from %s: %w", file, err)
		}
	}

	return nil
}

func (c *Client) loadAndApplyFile(ctx context.Context, filePath string, user string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var yamlTemplate YamlTemplate
	if err := yaml.Unmarshal(data, &yamlTemplate); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	if yamlTemplate.Service.Name == "" {
		return fmt.Errorf("service.id is required in template file")
	}

	for _, group := range yamlTemplate.Groups {
		if err := c.applyGroup(ctx, &yamlTemplate, &group, user); err != nil {
			return fmt.Errorf("failed to apply group %s: %w", group.ID, err)
		}
		log.Printf("Successfully imported template: service=%s, group=%s from %s",
			yamlTemplate.Service.Name, group.ID, filepath.Base(filePath))
	}

	return nil
}

func (c *Client) applyGroup(ctx context.Context, yamlTemplate *YamlTemplate, group *YamlGroup, user string) error {
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
			SortOrder:    f.SortOrder,
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
		SortOrder:        group.SortOrder,
	}

	_, err := c.ApplyConfigTemplate(ctx, template, user)
	return err
}

func toFieldType(s string) configv1.FieldType {
	switch s {
	case "STRING":
		return configv1.FieldType_STRING
	case "INT":
		return configv1.FieldType_INT
	case "FLOAT":
		return configv1.FieldType_FLOAT
	case "BOOLEAN":
		return configv1.FieldType_BOOLEAN
	case "JSON":
		return configv1.FieldType_JSON
	case "ARRAY_STRING":
		return configv1.FieldType_ARRAY_STRING
	case "SECRET":
		return configv1.FieldType_SECRET
	default:
		return configv1.FieldType_FIELD_TYPE_UNSPECIFIED
	}
}

func toScope(s string) configv1.Scope {
	switch s {
	case "SYSTEM":
		return configv1.Scope_SYSTEM
	case "PROJECT":
		return configv1.Scope_PROJECT
	case "STORE":
		return configv1.Scope_STORE
	case "USER":
		return configv1.Scope_USER
	default:
		return configv1.Scope_SCOPE_UNSPECIFIED
	}
}

// LoadTemplateFromBytes loads a single template from YAML bytes and applies it.
//
// Example:
//
//	yamlData := []byte(`
//	service:
//	  id: "my-service"
//	  label: "My Service"
//	groups:
//	  - id: "settings"
//	    label: "Settings"
//	    fields:
//	      - path: "log.level"
//	        label: "Log Level"
//	        type: "STRING"
//	        defaultValue: "INFO"
//	`)
//	err := client.LoadTemplateFromBytes(ctx, yamlData, "system")
func (c *Client) LoadTemplateFromBytes(ctx context.Context, data []byte, user string) error {
	var yamlTemplate YamlTemplate
	if err := yaml.Unmarshal(data, &yamlTemplate); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	if yamlTemplate.Service.Name == "" {
		return fmt.Errorf("service.id is required in template")
	}

	for _, group := range yamlTemplate.Groups {
		if err := c.applyGroup(ctx, &yamlTemplate, &group, user); err != nil {
			return fmt.Errorf("failed to apply group %s: %w", group.ID, err)
		}
	}

	return nil
}
