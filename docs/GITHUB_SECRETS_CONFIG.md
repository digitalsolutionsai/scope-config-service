# GitHub Secrets Configuration

## ✅ GCP Setup Complete

All GCP resources have been created and configured. Now you need to configure the GitHub environment and add secrets.

## Step 1: Create GitHub Environment (if not exists)

1. Go to: `https://github.com/digitalsolutionsai/scope-config-service/settings/environments`
2. If `production` environment doesn't exist, click "New environment"
3. Name it: `production`
4. Click "Configure environment"
5. (Optional) Add protection rules:
   - Required reviewers
   - Wait timer
   - Deployment branches (e.g., only `main` branch)

## Step 2: Add Environment Secrets

**Important:** The secrets must be added to the `production` environment, not as repository secrets.

Go to: `https://github.com/digitalsolutionsai/scope-config-service/settings/environments`

1. Click on the `production` environment (or create it if it doesn't exist)
2. Under "Environment secrets", click "Add secret" for each:

### 1. GCP_WORKLOAD_IDENTITY_PROVIDER
```
projects/939986339517/locations/global/workloadIdentityPools/github-pool/providers/github-provider
```

### 2. GCP_SERVICE_ACCOUNT
```
github-actions-sa@dsai-production.iam.gserviceaccount.com
```

## Test the Workflow

Once the secrets are added, test the workflow:

```bash
# Create a tag
git tag 2511.02.1-prd

# Push the tag
git push origin 2511.02.1-prd
```

The workflow will automatically:
1. Build the Docker image
2. Push to: `asia-southeast1-docker.pkg.dev/dsai-production/scope-config-service/scope-config-service:2511.02.1-prd`
3. Also tag as: `asia-southeast1-docker.pkg.dev/dsai-production/scope-config-service/scope-config-service:latest`

## What Was Created in GCP

✅ Service Account: `github-actions-sa@dsai-production.iam.gserviceaccount.com`
✅ Workload Identity Pool: `github-pool`
✅ Workload Identity Provider: `github-provider`
✅ Permissions: Artifact Registry Writer on `scope-config-service` repository
✅ Repository restriction: Only `digitalsolutionsai/scope-config-service` can use this identity

## Verify Images After Push

```bash
# List images in the repository
gcloud artifacts docker images list \
    asia-southeast1-docker.pkg.dev/dsai-production/scope-config-service/scope-config-service

# Pull the image
docker pull asia-southeast1-docker.pkg.dev/dsai-production/scope-config-service/scope-config-service:2511.02.1-prd
```

## Security Notes

- ✅ Using Workload Identity Federation (no long-lived credentials)
- ✅ Repository-specific permissions (not project-wide)
- ✅ Restricted to `digitalsolutionsai` organization only
- ✅ Specific repository access only
