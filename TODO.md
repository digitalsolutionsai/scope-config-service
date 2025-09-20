###  **Problem Summary (AS-IS)**

The `group_id` is a critical part of a configuration's identity but is not currently enforced by the system. This leads to inconsistent data, inefficient `COALESCE` logic in SQL queries, and an incomplete API. The goal is to make **`group_id` a mandatory part of every configuration's identity.**

-----

<br>

###  **Action Plan & Files to Modify**

Here is the precise checklist of files and the actions required, now with clearer examples.

#### **1. Database Migration 🐘**

  * **Files:** Create **two new migration files** in the `db/migrations/` directory.
      * `YYYYMMDDHHMMSS_enforce_required_group_id.up.sql`
          * **To-Do:** Alter `config_version` and `config_template` tables to make the `group_id` column `NOT NULL`. Add a composite index for performance.
      * `YYYYMMDDHHMMSS_enforce_required_group_id.down.sql`
          * **To-Do:** Add SQL to revert the changes.

#### **2. Server Logic ⚙️**

  * **Files:**
      * `pkg/service/config_handlers.go`
      * `pkg/service/template_handlers.go`
  * **To-Do:** In both files, find every SQL query and **remove all `COALESCE` logic**. Modify the queries to use `group_id` as a direct, required parameter.

#### **3. API & CLI Client 💻**

  * **File:** `proto/config/v1/config.proto`
      * **To-Do:** Update the comment for the `group_id` field to `// Required.` and run `make proto`.
  * **File:** `cmd/cli/main.go`
      * **To-Do:** Add the persistent `--group-id` flag, add validation to make it mandatory, and update the `createIdentifier` function.
  * **Files:**
      * `cmd/cli/get.go`
      * `cmd/cli/set.go`
      * `cmd/cli/publish.go`
      * `cmd/cli/show.go`
      * `cmd/cli/template.go`
  * **To-Do:** Update the `Example:` string in each of these files to include the required `--group-id` flag.

**Example for `set.go`:**

  * **Before:**
    `config-cli set --service-name=api --scope=PROJECT --project-id=proj_123 db.user=admin`
  * **After:**
    `config-cli set --service-name=api --scope=PROJECT --project-id=proj_123 --group-id=database db.user=admin`

**Example for `publish.go`:**

  * **Before:**
    `config-cli publish 2 --service-name=api --scope=PROJECT --project-id=proj_123`
  * **After:**
    `config-cli publish 2 --service-name=api --scope=PROJECT --project-id=proj_123 --group-id=database`

#### **4. Documentation 📖**

  * **File:** `README.md`
  * **To-Do:** Review and update all `config-cli` command examples. The new commands should clearly show `group_id` in action.

**Example workflow to add to the README:**

A user managing Stripe settings for the `billing-service` would now use the following commands:

1.  **Set** the configuration values for the `stripe` group:
    ```bash
    docker compose exec config-service config-cli set \
      --service-name=billing-service \
      --scope=PROJECT \
      --project-id=project-123 \
      --group-id=stripe \
      --user-name="John Doe" \
      apiKey=sk_test_... \
      apiVersion=2023-10-16
    ```
2.  **Show** the active vs. published versions for the `stripe` group:
    ```bash
    docker compose exec config-service config-cli show \
      --service-name=billing-service \
      --scope=PROJECT \
      --project-id=project-123 \
      --group-id=stripe
    ```
3.  **Publish** version 1 of the `stripe` group configuration:
    ```bash
    docker compose exec config-service config-cli publish 1 \
      --service-name=billing-service \
      --scope=PROJECT \
      --project-id=project-123 \
      --group-id=stripe \
      --user-name="John Doe"
    ```

-----

<br>

###  **⚠️ A Critical Warning on Changes**

This is a targeted **breaking change**. Your focus must be on implementing *only* the modifications listed above.

  * **STICK TO THE PLAN:** Do not use this opportunity to perform unrelated refactoring.
  * **DO NOT RENAME PACKAGES OR FUNCTIONS:** Avoid changing any existing package names, function names, or variable names unless it is a direct part of this task.
  * **DO NOT REMOVE FUNCTIONS:** No functions should be removed.

Introducing unrelated changes will significantly increase the risk of new bugs and make your work much harder to review and verify. Let's keep this change clean and focused.