# Template Fallback Feature

## Overview
The configuration service now supports template fallback values when configurations are not set or not published. This ensures that services always receive meaningful default values from templates instead of empty configurations.

## How It Works

### Scenarios Where Template Fallback Is Used

1. **No Configuration Exists**: When `GetConfig` is called for a service/group combination that has never been configured
2. **No Published Version**: When a configuration exists but no version has been published yet

### Fallback Logic

When either scenario occurs, the system will:

1. Check if a template exists for the requested `service_name` and `group_id`
2. Query template fields that have:
   - Non-null and non-empty `default_value`
   - Display scope that matches the request scope OR includes `SYSTEM` scope
3. Return the template default values as configuration fields
4. Set `CurrentVersion` to `0` to indicate template fallback is being used

### Single Path vs Full Group Requests

#### Single Path Request (`path` parameter specified)
- Returns only the template field matching the specified path
- Example: `GetConfig` with `path: "rules"` returns only the `rules` field default value

#### Full Group Request (`path` parameter empty or not specified)
- Returns ALL template fields for the group that have default values
- Example: `GetConfig` without `path` returns all template fields with defaults for that group
- CLI shows "Template Fallback" instead of "No configuration found"

### Database Query for Template Fallback

```sql
SELECT path, default_value 
FROM config_template_field 
WHERE config_template_id = ? 
AND default_value IS NOT NULL 
AND default_value != '' 
AND (? = ANY(display_on) OR 'SYSTEM' = ANY(display_on))
```

## Example Usage

### Template Data
```sql
INSERT INTO public.config_template
(id, service_name, service_label, group_id, group_label, group_description, updated_by, created_at, updated_at, created_by)
VALUES(13, 'api-gateway', 'API Gateway Service', 'cors-rules', 'CORS Configuration', 'Cross-Origin Resource Sharing rules and settings.', 'Loi Vo', '2025-09-29 23:31:57.395', '2025-09-29 23:31:57.395', 'Loi Vo');

INSERT INTO public.config_template_field
(id, config_template_id, "path", "label", description, "type", default_value, display_on, "options")
VALUES(41, 13, 'rules', 'CORS Rules', 'JSON configuration for CORS rules', 'JSON', '[{"path-pattern": "/**", "allowed-origins": ["http://localhost:4200"], "allowed-methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"], "allow-credentials": true, "allowed-headers": ["Authorization", "Content-Type"], "exposed-headers": [], "max-age": 3600}]', '{SYSTEM,PROJECT}', 'null'::jsonb);
```

### API Request
```protobuf
GetConfigRequest {
  identifier: {
    service_name: "api-gateway"
    group_id: "cors-rules"
    scope: PROJECT
    project_id: "my-project"
  }
  // path: ""  // Empty path = get full group
}
```

### Response When No Published Config Exists (Full Group)
```protobuf
ScopeConfig {
  version_info: {
    identifier: { ... }
    latest_version: 0
    published_version: null
  }
  current_version: 0  // Indicates template fallback
  fields: [
    {
      path: "rules"
      value: "[{\"path-pattern\": \"/**\", \"allowed-origins\": [\"http://localhost:4200\"], ...}]"
    },
    {
      path: "timeout"
      value: "30000"
    },
    {
      path: "rate-limit"
      value: "100"
    }
    // ... all other template fields with default values
  ]
}
```

### CLI Output for Full Group Template Fallback
```
Version:        Template Fallback
Status:         Using Template Defaults
Updated At:     N/A
Updated By:     

Template Default Fields:
  rules: [{"path-pattern": "/**", "allowed-origins": ["http://localhost:4200"], ...}]
  timeout: 30000
  rate-limit: 100
```

## Benefits

1. **Graceful Degradation**: Services receive sensible defaults instead of empty configurations
2. **Faster Development**: New services can start with template defaults without manual configuration
3. **Consistency**: All services of the same type start with the same baseline configuration
4. **Reduced Errors**: Less likely to have services fail due to missing configuration

## Implementation Details

### Modified Functions
- `getConfig()`: Enhanced with template fallback logic
- `getTemplateDefaultFields()`: New helper function to retrieve template defaults

### Key Changes
- Added `shouldUseTemplateFallback` flag for published version checking
- Template fields are filtered by scope permissions
- `CurrentVersion` is set to `0` to distinguish template fallback from real configurations

### Scope Filtering
Template fields are only returned if:
- The field's `display_on` array includes the requested scope, OR
- The field's `display_on` array includes `SYSTEM` (universal fallback)

This ensures that sensitive or scope-specific configurations are only returned to appropriate callers.