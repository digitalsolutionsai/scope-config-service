# HTTP Gateway Quick Start Guide

This guide will help you quickly get started with the HTTP Gateway for the Scope Configuration Service.

**Note:** The service now runs both gRPC and HTTP in a **single container**. The HTTP gateway internally connects to the local gRPC service on localhost.

## 🚀 Quick Start (Development Mode - No Auth)

For quick local testing without authentication:

```bash
# 1. Start the services (runs both gRPC and HTTP in one container)
docker compose -f compose.postgres.yml -f compose.yml up -d --build

# 2. Test the API (no authentication required in development mode)
curl -X GET "http://localhost:8080/api/v1/config/payment-service/template?groupId=stripe"
```

## 🔐 Production Setup (With API Gateway)

**Authentication is handled at the API Gateway level** (e.g., Spring Cloud Gateway, Kong, Nginx).

### Architecture

```
Client → API Gateway (Auth) → Config Service (Public APIs)
```

### Setup Steps

1. **Deploy Config Service** as shown in Quick Start above
2. **Configure API Gateway** to:
   - Handle JWT token validation
   - Enforce role-based access control
   - Route requests to config service
   - Optionally pass user info in headers

3. **Example API Gateway Configuration** (Spring Cloud Gateway):

```yaml
spring:
  cloud:
    gateway:
      routes:
        - id: config-service
          uri: http://config-service:8080
          predicates:
            - Path=/api/v1/config/**
          filters:
            - TokenRelay
            - name: RequestRateLimiter
```

4. **Client requests go through gateway**:

```bash
# Client calls API Gateway (with authentication)
curl -X GET "https://gateway.example.com/api/v1/config/payment-service/template?groupId=stripe" \
  -H "Authorization: Bearer $JWT_TOKEN"

# Gateway validates token and forwards to service
```

## 📝 Common API Examples

### 1. Get Template (For Building Dynamic UI Forms)

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment-service/template?groupId=stripe" \
 
```

**Use Case:** Frontend fetches this to know which fields to display and what types they are.

### 2. Get Current Published Configuration

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment-service/scope/PROJECT?groupId=stripe&projectId=proj-123" \
 
```

**Use Case:** Get the active configuration that's currently in use.

### 3. Get Latest Configuration (Including Unpublished)

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment-service/scope/PROJECT/latest?groupId=stripe&projectId=proj-123" \
 
```

**Use Case:** Preview changes that haven't been published yet.

### 4. Get Version History

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment-service/scope/PROJECT/history?groupId=stripe&projectId=proj-123&limit=10" \
 
```

**Use Case:** Show audit trail of who changed what and when.

### 5. Publish a Configuration Version

```bash
curl -X POST "http://localhost:8080/api/v1/config/payment-service/scope/PROJECT/publish?groupId=stripe" \
  \
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
  'http://localhost:8080/api/v1/config/payment-service/template?groupId=stripe',
  {
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

### "userName is required"

**Problem:** You didn't provide userName in the publish request.

**Solution:** Include `"userName": "user@example.com"` in the request body for audit trail purposes.

### Connection refused

**Problem:** The service is not running or not accessible.

**Solution:** 
1. Verify service is running: `docker compose ps`
2. Check service logs: `docker compose logs config-service`
3. Ensure ports 50051 (gRPC) and 8080 (HTTP) are not in use

### Template or config not found

**Problem:** The requested service/group doesn't exist in the database.

**Solution:** 
1. Apply templates using the CLI tool
2. Verify service name and groupId are correct
3. Check database has template records

## 📚 More Information

- **Full API Documentation**: [HTTP_GATEWAY.md](./HTTP_GATEWAY.md)
- **Main README**: [../README.md](../README.md)
- **API Gateway Setup**: See [Authentication section](./HTTP_GATEWAY.md#authentication) for gateway integration

## 💡 Tips

1. **Authentication at Gateway**: Configure authentication in your API Gateway (Spring, Kong, etc.)
2. **Direct Access**: Service APIs are public - use only behind an authenticated gateway in production
3. **Audit Trail**: Always provide userName in publish requests for proper audit logging
4. **Scope Hierarchy**: Remember that configs cascade: SYSTEM → PROJECT → STORE → USER

---

**Need Help?** Check the full [HTTP Gateway Documentation](./HTTP_GATEWAY.md) or create an issue on GitHub.
