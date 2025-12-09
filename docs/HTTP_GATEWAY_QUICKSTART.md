# HTTP Gateway Quick Start Guide

This guide will help you quickly get started with the HTTP Gateway for the Scope Configuration Service.

**Note:** The service now runs both gRPC and HTTP in a **single container**. The HTTP gateway internally connects to the local gRPC service on localhost.

## 🚀 Quick Start (Development Mode - No Auth)

For quick local testing without authentication:

```bash
# 1. Start the services (runs both gRPC and HTTP in one container)
docker compose -f compose.postgres.yml -f compose.yml up -d --build

# 2. Test the API (no authentication required in development mode)
curl -X GET "http://localhost:8080/api/v1/templates/payment-service?groupId=stripe"
```

## 🔐 Production Setup (With Keycloak)

### Step 1: Get Your Keycloak Public Key

1. Log into your Keycloak Admin Console
2. Navigate to: **Realm Settings** → **Keys** tab
3. Find the active **RS256** key (Status: Active)
4. Click the **Public key** button
5. Copy the base64 string that appears

### Step 2: Configure Environment Variables

Create or update your `.env` file:

```bash
# Copy from example
cp .env.example .env

# Edit .env and add:
KEYCLOAK_PUBLIC_KEY=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
KEYCLOAK_CLIENT=dsai-console
KEYCLOAK_ROLES=ROLE_ADMIN
```

### Step 3: Update Docker Compose

Edit `compose.yml` and uncomment the Keycloak variables:

```yaml
http-gateway:
  environment:
    - GRPC_SERVER_ADDRESS=config-service:50051
    - HTTP_PORT=8080
    - KEYCLOAK_PUBLIC_KEY=${KEYCLOAK_PUBLIC_KEY}
    - KEYCLOAK_CLIENT=dsai-console
    - KEYCLOAK_ROLES=ROLE_ADMIN
```

### Step 4: Start the Services

```bash
docker compose -f compose.postgres.yml -f compose.yml up -d --build
```

### Step 5: Get a JWT Token

Log in through your Keycloak login page or use the Keycloak API to get a token:

```bash
# Example using Keycloak token endpoint
curl -X POST "https://auth.dsai.vn/realms/sso/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=dsai-console" \
  -d "username=your.email@dsai.vn" \
  -d "password=your-password" \
  -d "grant_type=password"
```

Extract the `access_token` from the response.

### Step 6: Make Authenticated Requests

```bash
# Store your token
export TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

# Make authenticated request
curl -X GET "http://localhost:8080/api/v1/templates/payment-service?groupId=stripe" \
  -H "Authorization: Bearer $TOKEN"
```

## 📝 Common API Examples

### 1. Get Template (For Building Dynamic UI Forms)

```bash
curl -X GET "http://localhost:8080/api/v1/templates/payment-service?groupId=stripe" \
  -H "Authorization: Bearer $TOKEN"
```

**Use Case:** Frontend fetches this to know which fields to display and what types they are.

### 2. Get Current Published Configuration

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment-service/scope/PROJECT?groupId=stripe&projectId=proj-123" \
  -H "Authorization: Bearer $TOKEN"
```

**Use Case:** Get the active configuration that's currently in use.

### 3. Get Latest Configuration (Including Unpublished)

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment-service/scope/PROJECT/latest?groupId=stripe&projectId=proj-123" \
  -H "Authorization: Bearer $TOKEN"
```

**Use Case:** Preview changes that haven't been published yet.

### 4. Get Version History

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment-service/scope/PROJECT/history?groupId=stripe&projectId=proj-123&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

**Use Case:** Show audit trail of who changed what and when.

### 5. Publish a Configuration Version

```bash
curl -X POST "http://localhost:8080/api/v1/config/payment-service/scope/PROJECT/publish?groupId=stripe" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "version": 5,
    "projectId": "proj-123"
  }'
```

**Use Case:** Make a specific version the active configuration. Note: `userName` is optional when authenticated - it will automatically use the email from your JWT token!

## 🎯 Frontend Integration Example

Here's a typical workflow for a configuration management UI:

```javascript
// 1. Fetch template to build the form
const template = await fetch(
  'http://localhost:8080/api/v1/templates/payment-service?groupId=stripe',
  {
    headers: { 'Authorization': `Bearer ${token}` }
  }
).then(r => r.json());

// 2. Render form based on template fields
template.fields.forEach(field => {
  if (field.type === 'STRING') {
    renderTextInput(field.path, field.label, field.defaultValue);
  } else if (field.type === 'BOOLEAN') {
    renderCheckbox(field.path, field.label, field.defaultValue === 'true');
  } else if (field.options.length > 0) {
    renderDropdown(field.path, field.label, field.options);
  }
});

// 3. Fetch current values
const config = await fetch(
  'http://localhost:8080/api/v1/config/payment-service/scope/PROJECT?groupId=stripe&projectId=proj-123',
  {
    headers: { 'Authorization': `Bearer ${token}` }
  }
).then(r => r.json());

// 4. Pre-populate form with current values
Object.entries(config.fields).forEach(([path, value]) => {
  setFormValue(path, value);
});

// 5. After user edits and saves, publish the new version
// (Note: You'd call the gRPC UpdateConfig first to create a new version,
//  then publish it. UpdateConfig is not yet exposed via HTTP.)
await fetch(
  'http://localhost:8080/api/v1/config/payment-service/scope/PROJECT/publish?groupId=stripe',
  {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      version: newVersion,
      projectId: 'proj-123'
    })
  }
);
```

## 🔧 Troubleshooting

### "missing Authorization header"

**Problem:** You forgot to include the Authorization header.

**Solution:** Add `-H "Authorization: Bearer $TOKEN"` to your curl command.

### "invalid token: token signature is invalid"

**Problem:** The public key doesn't match the token, or the token is from a different realm.

**Solution:** 
1. Verify you copied the correct public key from the right realm
2. Make sure the token is from the same Keycloak realm
3. Check that the token hasn't expired

### "insufficient permissions"

**Problem:** Your user doesn't have the `ROLE_ADMIN` role in the `dsai-console` client.

**Solution:** In Keycloak Admin Console:
1. Go to **Users** → Find your user
2. Go to **Role Mappings** tab
3. Select `dsai-console` from **Client Roles** dropdown
4. Assign `ROLE_ADMIN` to the user
5. Get a new token

### "userName is required when authentication is disabled"

**Problem:** You're running without authentication and didn't provide userName in the publish request.

**Solution:** Either:
- Enable authentication by setting `KEYCLOAK_PUBLIC_KEY`
- OR include `"userName": "your-email@example.com"` in the request body

## 📚 More Information

- **Full API Documentation**: [HTTP_GATEWAY.md](./HTTP_GATEWAY.md)
- **Main README**: [../README.md](../README.md)
- **Keycloak Setup**: See [Authentication section](./HTTP_GATEWAY.md#authentication) in the full documentation

## 💡 Tips

1. **Save Your Token**: Store the JWT token in a variable for easy reuse
2. **Check Token Expiry**: JWT tokens expire (typically after 1 hour). Get a new one when needed.
3. **Use .env for Keys**: Never commit your `KEYCLOAK_PUBLIC_KEY` to git
4. **Test in Development Mode**: Start without auth for initial testing, then add security
5. **Scope Hierarchy**: Remember that configs cascade: SYSTEM → PROJECT → STORE → USER

---

**Need Help?** Check the full [HTTP Gateway Documentation](./HTTP_GATEWAY.md) or create an issue on GitHub.
