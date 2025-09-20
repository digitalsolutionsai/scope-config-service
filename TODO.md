
# TODO List

- [ ] **Proto Files:**
    - [ ] Update `proto/config/v1/config.proto`:
        - [ ] Remove `service_name` from the `ConfigIdentifier` message.
        - [ ] Update the `Scope` enum to `SYSTEM`, `PROJECT`, `STORE`, `USER`.

- [ ] **Database:**
    - [ ] Create a new migration script to update the database schema.
        - [ ] Remove the `service_name` column from the `configs` table.
        - [ ] Modify the `scope` column to be an `enum` with the values `SYSTEM`, `PROJECT`, `STORE`, `USER`.
        - [ ] Ensure `scope_value` is not nullable and defaults to "default" when `scope` is `SYSTEM`.
        - [ ] Update queries to be case-sensitive for `path` and `scope_id`.

- [ ] **Backend (Go):**
    - [ ] Update `pkg/service/config.go` and `pkg/service/config_handlers.go` to align with the new schema.
    - [ ] Remove all references to `serviceName` from the service logic.
    - [ ] Update `server.go` to remove any `serviceName` dependencies.

- [ ] **CLI:**
    - [ ] Update `cmd/cli/main.go`:
        - [ ] Remove the `serviceName` flag.
        - [ ] Update the help text for the `scope` flag.
    - [ ] Update `cmd/cli/get.go`, `cmd/cli/publish.go`, `cmd/cli/set.go`, and `cmd/cli/show.go` to remove `serviceName` usage.

- [ ] **Templates:**
    - [ ] Update `templates/example.yaml` and `templates/notification.yaml` to remove the `serviceName` field.

- [ ] **Documentation:**
    - [ ] Update `blueprint.md` and `README.md` to reflect the changes.
