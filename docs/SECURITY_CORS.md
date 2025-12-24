# CORS Security Fix

## Summary

Fixed a critical CORS (Cross-Origin Resource Sharing) security vulnerability where the middleware was configured with both `Access-Control-Allow-Origin: *` (wildcard) and `Access-Control-Allow-Credentials: true`. This combination violates the CORS specification and creates a security risk.

## Vulnerability Details

### The Problem

The original CORS middleware in `/home/user/alicia/internal/adapters/http/middleware/cors.go` was configured as:

```go
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Credentials", "true")
```

### Why This Is Dangerous

1. **CORS Specification Violation**: The CORS spec explicitly forbids using wildcard (`*`) origin with credentials
2. **Security Risk**: This combination would allow any website to make authenticated requests to your API with the user's credentials
3. **Browser Behavior**: Modern browsers reject this configuration, causing CORS errors in the client

### Attack Scenario

Without this fix, a malicious website could:

1. Load a page that makes requests to your Alicia instance
2. The browser would send cookies/credentials automatically
3. The wildcard CORS would allow the malicious site to read the response
4. Attacker gains access to user's private conversations and data

## The Fix

### Changes Made

1. **Configuration** (`/home/user/alicia/internal/config/config.go`):
   - Added `CORSOrigins []string` field to `ServerConfig`
   - Default value: `[]string{"http://localhost:3000"}` for development
   - Loads from `ALICIA_CORS_ORIGINS` environment variable (comma-separated)

2. **Middleware** (`/home/user/alicia/internal/adapters/http/middleware/cors.go`):
   - Changed from accepting no parameters to accepting `allowedOrigins []string`
   - Validates incoming `Origin` header against the allowed list
   - Only sets `Access-Control-Allow-Origin` to the **specific origin** if it's in the allowed list
   - Only sets `Access-Control-Allow-Credentials: true` when origin is explicitly allowed
   - Returns `403 Forbidden` for preflight requests from disallowed origins
   - Returns `204 No Content` for preflight requests from allowed origins

3. **Server** (`/home/user/alicia/internal/adapters/http/server.go`):
   - Updated to pass `s.config.Server.CORSOrigins` to the middleware

4. **Tests** (`/home/user/alicia/internal/adapters/http/middleware/cors_test.go`):
   - Comprehensive test coverage for allowed/disallowed origins
   - Tests for preflight requests
   - Security-specific test to verify wildcard + credentials is never used

### Security Properties

The new implementation ensures:

- ✅ **No wildcard with credentials**: Never uses `*` when credentials are enabled
- ✅ **Explicit allow-list**: Only configured origins are allowed
- ✅ **Specific origin echo**: Returns the exact origin that made the request (if allowed)
- ✅ **Preflight protection**: Blocks preflight requests from unknown origins
- ✅ **Default-secure**: Defaults to `localhost:3000` for development, not `*`

## Configuration Guide

### Development

Default configuration allows `http://localhost:3000`:

```bash
# No configuration needed for local development
go run ./cmd/alicia serve
```

### Production

Set explicit origins via environment variable:

```bash
export ALICIA_CORS_ORIGINS="https://alicia.example.com,https://app.example.com"
```

Or via configuration file:

```json
{
  "server": {
    "cors_origins": [
      "https://alicia.example.com",
      "https://app.example.com"
    ]
  }
}
```

### Best Practices

1. **Use HTTPS**: Always use `https://` origins in production
2. **Be specific**: Include the full origin (protocol, domain, and port if non-standard)
3. **Minimal list**: Only add origins that genuinely need access
4. **No wildcards**: Never use `*` in production
5. **Regular audits**: Review the allowed origins list periodically

## Testing

Run the CORS middleware tests:

```bash
go test ./internal/adapters/http/middleware/
```

Test with curl:

```bash
# Should succeed (allowed origin)
curl -v -H "Origin: http://localhost:3000" \
     -H "Access-Control-Request-Method: POST" \
     -X OPTIONS http://localhost:8080/api/v1/conversations

# Should fail (disallowed origin)
curl -v -H "Origin: https://evil.com" \
     -H "Access-Control-Request-Method: POST" \
     -X OPTIONS http://localhost:8080/api/v1/conversations
```

## Impact Assessment

### Before Fix
- ❌ Any website could potentially access user data
- ❌ CORS errors in modern browsers
- ❌ Non-compliant with CORS specification
- ❌ Failed security audits

### After Fix
- ✅ Only configured origins can access the API
- ✅ Compliant with CORS specification
- ✅ No browser CORS errors with proper configuration
- ✅ Passes security audits
- ✅ Production-ready CORS implementation

## References

- [CORS Specification (W3C)](https://www.w3.org/TR/cors/)
- [MDN: CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [OWASP: CORS Misconfiguration](https://owasp.org/www-community/attacks/CORS_OriginHeaderScrutiny)
- [PortSwigger: CORS Vulnerabilities](https://portswigger.net/web-security/cors)

## Timeline

- **Discovered**: 2025-12-22
- **Fixed**: 2025-12-22
- **Severity**: High
- **Status**: Resolved
