### Phase 1: Update Core Contracts & Data Structures

This is the most critical phase. Changes here will cause compilation errors that will guide the rest of the refactoring.

  * **1.1. Refactor Protobuf Definitions (`proto/config/v1/config.proto`)**

      * In the `ConfigTemplate` message, add the new descriptive fields:
        ```proto
        message ConfigTemplate {
            ConfigIdentifier identifier = 1;
            string service_label = 2;       // ADD THIS
            string group_label = 3;         // ADD THIS
            string group_description = 4;   // ADD THIS
            repeated ConfigFieldTemplate fields = 5;
        }
        ```
      * In the `ConfigField` message, remove the fields that were moved to the template:
        ```proto
        message ConfigField {
            string path = 1;
            string value = 2;
            FieldType type = 3;
            // string default_value = 4; // REMOVE THIS
            // string description = 5;   // REMOVE THIS
        }
        ```
      * Redefine the `GetConfigHistory` response to match the new `config_version_history` table for a true audit log:
          * Create a new message for a single history entry:
            ```proto
            message VersionHistoryEntry {
                int32 version = 1;
                google.protobuf.Timestamp created_at = 2;
                string created_by = 3;
            }
            ```
          * Update the `GetConfigHistoryResponse` message:
            ```proto
            message GetConfigHistoryResponse {
                // repeated ConfigVersion versions = 1; // REPLACE THIS
                repeated VersionHistoryEntry history = 1; // WITH THIS
            }
            ```

  * **1.2. Regenerate Protobuf Code**

      * Run your script (`./scripts/gen-proto.sh`) to update the generated Go files (`*.pb.go`, `*_grpc.pb.go`). This will break the build, which is expected.

-----

### Phase 2: Implement Backend Service Logic

Update the gRPC handlers in `pkg/service/` to work with the new database schema and Protobuf messages.

  * **2.1. Update Template Handlers (`pkg/service/template_handlers.go`)**

      * **`ApplyConfigTemplate`:**
          * Modify the `upsertQuery` to insert the new label and description fields.
          * Update the `tx.QueryRowContext` call to pass `template.ServiceLabel`, `template.GroupLabel`, and `template.GroupDescription` as arguments.
      * **`GetConfigTemplate`:**
          * Modify the `SELECT` query to retrieve `service_label`, `group_label`, and `group_description` from the `config_template` table.
          * Scan these new columns and populate them into the returned `ConfigTemplate` response object.

  * **2.2. Update Config Handlers (`pkg/service/config_handlers.go`)**

      * **`UpdateConfig`:**
          * Within the database transaction, after successfully updating the `config_version` table, add a new SQL statement to **insert an entry into `config_version_history`**.
          * The new query should be: `INSERT INTO config_version_history (config_version_id, version, created_by) VALUES ($1, $2, $3)` using the `configVersionID`, `newVersion`, and `req.User`.
      * **`GetConfigHistory`:**
          * **Completely rewrite this function.** The old logic queries the wrong table.
          * First, query `config_version` to get the primary `id` for the given identifier.
          * Second, use that `id` to query `config_version_history`: `SELECT version, created_at, created_by FROM config_version_history WHERE config_version_id = $1 ORDER BY version DESC`.
          * Iterate through the results and populate the new `repeated VersionHistoryEntry` in the `GetConfigHistoryResponse`.

-----

### Phase 3: Adapt CLI and Template Files

Modify the command-line interface and the YAML templates to support the new, richer data model.

  * **3.1. Refactor Template CLI (`cmd/cli/template.go`)**

      * **Update YAML Parsing:** The current code unmarshals the entire file into a single `ConfigTemplate`, which is now incorrect.
      * Define new Go structs that mirror the new YAML structure (`service`, `groups` list).
      * **Change `applyCmd` Logic:**
          * Parse the YAML file into these new structs.
          * Loop through each `group` in the parsed file.
          * Inside the loop, for each group, construct a `configv1.ApplyConfigTemplateRequest` and make a separate gRPC call to `ApplyConfigTemplate`. This ensures each group's template is applied individually.

      * 3.2 Implement the `history` Command and Refactor `show`

        This task is now split into two parts: creating the new `history` command and cleaning up the old `show` command.

        **Part A: Create New `history` Command (`cmd/cli/history.go`)**

        * Create a new file named `cmd/cli/history.go`.
        * Inside this file, define a **new Cobra command** called `historyCmd`.
        * The command's `Run` function should:
            * Call the `GetConfigHistory` gRPC endpoint.
            * Adapt the logic to handle the new `GetConfigHistoryResponse` with its `repeated VersionHistoryEntry`.
            * Update the `tabwriter` output to display the new, simpler history columns: **Version**, **Created At**, and **Created By**.
        * Register the new `historyCmd` in `cmd/cli/main.go`.

        **Part B: Clean Up `show` Command (`cmd/cli/show.go`)**

        * **Remove all history-related logic** from the `showCmd`.
        * Delete the `showHistory` function entirely.
        * Remove the `--history` and `--limit` flags from the command's `init()` function.
        * Ensure the `showCmd`'s `Run` function now only handles fetching and displaying the latest and published configurations.
