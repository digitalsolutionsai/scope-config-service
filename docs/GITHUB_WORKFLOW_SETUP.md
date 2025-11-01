# GitHub Workflow Setup Guide for Google Cloud Artifact Registry

This guide will help you set up the necessary credentials and configurations to automatically build and push Docker images to Google Cloud Artifact Registry using GitHub Actions.

## Overview

The workflow automatically builds and pushes Docker images when you create a tag ending with `-prd` (e.g., `2511.02.1-prd`).

**Image URL:** `asia-southeast1-docker.pkg.dev/dsai-production/scope-config-service/scope-config-service`

## Prerequisites

- Access to the `dsai-production` GCP project
- GitHub repository admin access
- `gcloud` CLI installed on your local machine

## Setup Steps

### 1. Create a Google Cloud Service Account

```bash
# Set your project
gcloud config set project dsai-production

# Create a service account
gcloud iam service-accounts create github-actions-sa \
    --display-name="GitHub Actions Service Account" \
    --description="Service account for GitHub Actions to push images to Artifact Registry"
```

### 2. Grant Artifact Registry Permissions

```bash
# Grant Artifact Registry Writer role
gcloud projects add-iam-policy-binding dsai-production \
    --member="serviceAccount:github-actions-sa@dsai-production.iam.gserviceaccount.com" \
    --role="roles/artifactregistry.writer"
```

### 3. Set Up Workload Identity Federation (Recommended Method)

Workload Identity Federation is more secure than service account keys as it doesn't require storing long-lived credentials.

#### 3.1. Create Workload Identity Pool

```bash
# Create the pool
gcloud iam workload-identity-pools create "github-pool" \
    --project="dsai-production" \
    --location="global" \
    --display-name="GitHub Actions Pool"

# Get the pool ID
gcloud iam workload-identity-pools describe "github-pool" \
    --project="dsai-production" \
    --location="global" \
    --format="value(name)"
```

#### 3.2. Create Workload Identity Provider

Replace `YOUR_GITHUB_ORG` and `YOUR_REPO_NAME` with your actual values:

```bash
# Create the provider
gcloud iam workload-identity-pools providers create-oidc "github-provider" \
    --project="dsai-production" \
    --location="global" \
    --workload-identity-pool="github-pool" \
    --display-name="GitHub Provider" \
    --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository,attribute.repository_owner=assertion.repository_owner" \
    --attribute-condition="assertion.repository_owner == 'YOUR_GITHUB_ORG'" \
    --issuer-uri="https://token.actions.githubusercontent.com"
```

#### 3.3. Allow GitHub to Impersonate the Service Account

```bash
# Allow the specific repository to impersonate the service account
gcloud iam service-accounts add-iam-policy-binding \
    "github-actions-sa@dsai-production.iam.gserviceaccount.com" \
    --project="dsai-production" \
    --role="roles/iam.workloadIdentityUser" \
    --member="principalSet://iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/github-pool/attribute.repository/YOUR_GITHUB_ORG/YOUR_REPO_NAME"
```

**Note:** Replace `PROJECT_NUMBER` with your GCP project number. Get it with:
```bash
gcloud projects describe dsai-production --format="value(projectNumber)"
```

#### 3.4. Get the Workload Identity Provider Name

```bash
gcloud iam workload-identity-pools providers describe "github-provider" \
    --project="dsai-production" \
    --location="global" \
    --workload-identity-pool="github-pool" \
    --format="value(name)"
```

This will output something like:
```
projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/github-pool/providers/github-provider
```

### 4. Configure GitHub Secrets

Go to your GitHub repository: `Settings` → `Secrets and variables` → `Actions` → `New repository secret`

Add the following secrets:

1. **GCP_WORKLOAD_IDENTITY_PROVIDER**
   - Value: The full provider name from step 3.4
   - Example: `projects/123456789/locations/global/workloadIdentityPools/github-pool/providers/github-provider`

2. **GCP_SERVICE_ACCOUNT**
   - Value: `github-actions-sa@dsai-production.iam.gserviceaccount.com`

### 5. Ensure Artifact Registry Repository Exists

```bash
# Check if the repository exists
gcloud artifacts repositories describe scope-config-service \
    --location=asia-southeast1 \
    --project=dsai-production

# If it doesn't exist, create it
gcloud artifacts repositories create scope-config-service \
    --repository-format=docker \
    --location=asia-southeast1 \
    --project=dsai-production \
    --description="Docker repository for scope-config-service"
```

## Usage

### Creating and Pushing a Tag

1. **Create a tag locally:**
   ```bash
   git tag 2511.02.1-prd
   ```

2. **Push the tag to GitHub:**
   ```bash
   git push origin 2511.02.1-prd
   ```

3. **Monitor the workflow:**
   - Go to your GitHub repository
   - Click on "Actions" tab
   - Watch the "Build and Push Docker Image" workflow

### Tag Format

The workflow is triggered by tags ending with `-prd`:
- ✅ `2511.02.1-prd`
- ✅ `v1.0.0-prd`
- ✅ `release-2511.02.1-prd`
- ❌ `2511.02.1` (no -prd suffix)
- ❌ `2511.02.1-dev` (wrong suffix)

### Resulting Images

When you push tag `2511.02.1-prd`, two images are created:

1. **Tagged version:**
   ```
   asia-southeast1-docker.pkg.dev/dsai-production/scope-config-service/scope-config-service:2511.02.1-prd
   ```

2. **Latest:**
   ```
   asia-southeast1-docker.pkg.dev/dsai-production/scope-config-service/scope-config-service:latest
   ```

## Pulling the Image

```bash
# Authenticate Docker to GCP
gcloud auth configure-docker asia-southeast1-docker.pkg.dev

# Pull the image
docker pull asia-southeast1-docker.pkg.dev/dsai-production/scope-config-service/scope-config-service:2511.02.1-prd
```

## Alternative: Using Service Account Key (Less Secure)

If you prefer using a service account key instead of Workload Identity Federation:

### 1. Create a Service Account Key

```bash
gcloud iam service-accounts keys create github-actions-key.json \
    --iam-account=github-actions-sa@dsai-production.iam.gserviceaccount.com
```

### 2. Add to GitHub Secrets

- **GCP_CREDENTIALS**
  - Value: The entire content of `github-actions-key.json`

### 3. Update Workflow

Replace the authentication step in `.github/workflows/build-and-push.yml`:

```yaml
- name: Authenticate to Google Cloud
  uses: google-github-actions/auth@v2
  with:
    credentials_json: ${{ secrets.GCP_CREDENTIALS }}
```

**⚠️ Warning:** Service account keys are long-lived credentials. If compromised, they can be used until revoked. Workload Identity Federation is the recommended approach.

## Troubleshooting

### Permission Denied Errors

```bash
# Verify service account has correct permissions
gcloud projects get-iam-policy dsai-production \
    --flatten="bindings[].members" \
    --filter="bindings.members:github-actions-sa@dsai-production.iam.gserviceaccount.com"
```

### Repository Not Found

```bash
# List all Artifact Registry repositories
gcloud artifacts repositories list --location=asia-southeast1 --project=dsai-production
```

### Workflow Not Triggering

- Ensure tag ends with `-prd`
- Check GitHub Actions is enabled for your repository
- Verify workflow file is on the branch/tag being pushed

## Security Best Practices

1. ✅ Use Workload Identity Federation instead of service account keys
2. ✅ Grant minimum required permissions (Artifact Registry Writer only)
3. ✅ Use attribute conditions to restrict which repositories can use the identity
4. ✅ Regularly review and audit service account usage
5. ✅ Use GitHub repository secrets (never commit credentials)

## Next Steps

After setup:
1. Test the workflow by creating and pushing a test tag
2. Verify images appear in Artifact Registry console
3. Update your deployment processes to use the new image location
4. Consider adding additional workflows for development/staging environments
