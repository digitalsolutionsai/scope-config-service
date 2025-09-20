# TODO List

- [ ] **Proto Files:**
    - [ ] Update `proto/config/v1/config.proto`:
        - [ ] Ensure `service_name` is a required field in the `ConfigIdentifier` message.
        - [ ] Update the `Scope` enum to `SYSTEM`, `PROJECT`, `STORE`, `USER`.
        - [ ] Add `user_id` to the `ConfigIdentifier` message.
        - [ ] Add comments detailing the length constraints for `project_id` (20), `store_id` (20), and `user_id` (35).

- [ ] **Database:**
    - [ ] Create a new migration script (`db/migrations/20250917100000_update_config_schema.up.sql`) to adjust the schema:
        - [ ] The `config_version` table needs a `scope_id` column to store project, store, or user IDs.
        - [ ] The `scope` column type should be updated to an `enum` (`SYSTEM`, `PROJECT`, `STORE`, `USER`).
        - [ ] When `scope` is `SYSTEM`, `scope_id` must be 'default'.
        - [ ] `service_name` remains a `NOT NULL` part of the primary key.
        - [ ] Ensure queries on `path` and `scope_id` are case-sensitive.

- [ ] **Backend (Go):**
    - [ ] Regenerate Go code from the updated `.proto` file.
    - [ ] Update `pkg/service/config.go` and `pkg/service/config_handlers.go` to handle the new `ConfigIdentifier` structure.
    - [ ] Implement logic to map the correct ID (`project_id`, `store_id`, `user_id`) to the `scope_id` database column based on the `scope` field.

- [ ] **CLI:**
    - [ ] Review all `cmd/cli/*.go` files to ensure they correctly use `service_name` as a required parameter and handle the updated `scope` options.

- [ ] **Documentation:**
    - [ ] Update `blueprint.md` and `README.md` to match the refined configuration identification strategy.
