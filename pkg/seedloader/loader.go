package seedloader

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"gopkg.in/yaml.v2"
)

// YamlValueOption represents a value option in YAML format
type YamlValueOption struct {
	Value string `yaml:"value"`
	Label string `yaml:"label"`
}

// YamlFieldTemplate represents a field template in YAML format
type YamlFieldTemplate struct {
	Path         string            `yaml:"path"`
	Label        string            `yaml:"label"`
	Description  string            `yaml:"description"`
	Type         string            `yaml:"type"`
	DefaultValue string            `yaml:"defaultValue"`
	DisplayOn    []string          `yaml:"displayOn"`
	Options      []YamlValueOption `yaml:"options"`
}

// YamlGroup represents a configuration group in YAML format
type YamlGroup struct {
	ID          string              `yaml:"id"`
	Label       string              `yaml:"label"`
	Description string              `yaml:"description"`
	Fields      []YamlFieldTemplate `yaml:"fields"`
}

// YamlTemplate represents the complete template structure in YAML format
type YamlTemplate struct {
	Service struct {
		Name  string `yaml:"id"`
		Label string `yaml:"label"`
	} `yaml:"service"`
	Groups []YamlGroup `yaml:"groups"`
}

// TemplateApplier is an interface for applying templates to the database
type TemplateApplier interface {
	ApplyConfigTemplate(ctx context.Context, req *configv1.ApplyConfigTemplateRequest) (*configv1.ConfigTemplate, error)
}

// Loader handles loading seed templates from a directory
type Loader struct {
	seedDir string
	applier TemplateApplier
}

// NewLoader creates a new template loader
func NewLoader(seedDir string, applier TemplateApplier) *Loader {
	return &Loader{
		seedDir: seedDir,
		applier: applier,
	}
}

// LoadAndApplyAll loads all YAML templates from the seed directory recursively and applies them
func (l *Loader) LoadAndApplyAll(ctx context.Context) error {
	if _, err := os.Stat(l.seedDir); os.IsNotExist(err) {
		log.Printf("Seed templates directory %s does not exist, skipping template import", l.seedDir)
		return nil
	}

	var templateFiles []string
	err := filepath.WalkDir(l.seedDir, func(path string, d fs.DirEntry, err error) error {
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
		return fmt.Errorf("failed to walk seed templates directory: %w", err)
	}

	if len(templateFiles) == 0 {
		log.Printf("No template files found in %s", l.seedDir)
		return nil
	}

	log.Printf("Found %d template file(s) to import", len(templateFiles))

	for _, file := range templateFiles {
		if err := l.loadAndApplyFile(ctx, file); err != nil {
			return fmt.Errorf("failed to apply template from %s: %w", file, err)
		}
	}

	return nil
}

func (l *Loader) loadAndApplyFile(ctx context.Context, filePath string) error {
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
		if err := l.applyGroup(ctx, &yamlTemplate, &group); err != nil {
			return fmt.Errorf("failed to apply group %s: %w", group.ID, err)
		}
		log.Printf("Successfully imported template: service=%s, group=%s from %s",
			yamlTemplate.Service.Name, group.ID, filepath.Base(filePath))
	}

	return nil
}

func (l *Loader) applyGroup(ctx context.Context, yamlTemplate *YamlTemplate, group *YamlGroup) error {
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
		User:     "system",
	}

	_, err := l.applier.ApplyConfigTemplate(ctx, req)
	return err
}

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
