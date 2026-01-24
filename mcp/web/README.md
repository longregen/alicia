# MCP Web

Web browsing and content extraction service. Fetches pages, converts to markdown, captures screenshots, and extracts structured data with SSRF protection and headless browser support.

## Tools

| Tool | Description |
|------|-------------|
| `read` | Fetch a URL and return clean markdown (best for articles/docs) |
| `fetch_raw` | Raw HTTP response with headers (best for APIs) |
| `fetch_structured` | Extract data using CSS selectors |
| `search` | Web search via DuckDuckGo or Kagi |
| `extract_links` | Extract and filter links from a page |
| `extract_metadata` | Extract page metadata (OG, Twitter, JSON-LD) |
| `screenshot` | Capture page screenshots using headless Chromium |

### `read`

Fetch a web page and convert to clean, LLM-friendly markdown using Mozilla Readability.

**Parameters:**
- `url` (string, required) - URL to fetch
- `include_links` (bool) - Keep links in output
- `include_images` (bool) - Keep image references
- `max_length` (int, default: 50000) - Max content length
- `wait_for` (string) - JS rendering wait: `none`, `load`, `domcontentloaded`, `networkidle`
- `wait_ms` (int) - Additional wait time in ms

```json
{
  "name": "read",
  "arguments": {
    "url": "https://example.com/article",
    "include_links": true,
    "wait_for": "networkidle"
  }
}
```

### `fetch_raw`

Get raw HTTP responses with full headers.

**Parameters:**
- `url` (string, required)
- `method` (string, default: GET) - GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS
- `headers` (object) - Custom headers
- `body` (string) - Request body
- `follow_redirects` (bool, default: true)

```json
{
  "name": "fetch_raw",
  "arguments": {
    "url": "https://api.example.com/data",
    "headers": { "Authorization": "Bearer token123" }
  }
}
```

### `fetch_structured`

Extract data using CSS selectors (powered by goquery).

**Parameters:**
- `url` (string, required)
- `selectors` (object, required) - Map of field names to CSS selectors
- `extract` (string, default: text) - `text`, `html`, or attribute name

```json
{
  "name": "fetch_structured",
  "arguments": {
    "url": "https://example.com/product",
    "selectors": { "title": "h1", "price": ".price", "links": "a[href]" }
  }
}
```

### `search`

Web search via DuckDuckGo (default) or Kagi (if `KAGI_API_KEY` is set).

**Parameters:**
- `query` (string, required, max 500 chars)
- `num_results` (int, 1-10, default: 5)
- `fetch_content` (bool) - Fetch and convert each result to markdown (truncated to 5000 chars)

### `extract_links`

Extract and filter links from a page.

**Parameters:**
- `url` (string, required)
- `filter` (string) - `all`, `internal`, or `external`
- `pattern` (string) - Regex filter for URLs
- `include_text` (bool) - Include anchor text
- `max_results` (int, default: 100)

### `extract_metadata`

Extract comprehensive page metadata: title, description, Open Graph, Twitter Cards, JSON-LD, favicons, canonical URL, publication dates.

**Parameters:**
- `url` (string, required)

### `screenshot`

Capture page screenshots using headless Chromium via Rod.

**Parameters:**
- `url` (string, required)
- `width` (int, default: 1280) - Viewport width
- `height` (int, default: 720) - Viewport height
- `full_page` (bool) - Capture full scrollable page
- `wait_for` (string) - Wait strategy before capture
- `wait_ms` (int, default: 1000)

Returns base64-encoded PNG.

## SSRF Protection

Multi-layered defense in `security/validator.go`:

1. **Scheme validation** - Only `http` and `https`
2. **Hostname blocklist** - localhost, .local, .internal, cloud metadata endpoints (AWS 169.254.169.254, GCP metadata.google.internal, Azure, Kubernetes)
3. **Private IP detection** - Loopback, private ranges (10.x, 172.16.x, 192.168.x), link-local, multicast, IPv4-mapped IPv6
4. **DNS resolution** - Resolves hostnames, validates all resolved IPs
5. **Redirect validation** - Each redirect target is validated

Applied to HTTP fetches, browser navigations, and screenshot captures.

## Headless Browser

Uses [Rod](https://go-rod.github.io/) for Chromium automation:

- Singleton browser pool with lazy initialization
- Pages created/closed per-request for isolation
- Wait strategies: `load`, `domcontentloaded`, `networkidle`
- Configurable viewport for rendering and screenshots

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `KAGI_API_KEY` | Kagi search API key (uses DuckDuckGo if unset) | - |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector endpoint | `https://alicia-data.hjkl.lol` |
| `ENVIRONMENT` | Environment label for telemetry | - |

## Resource Limits

| Limit | Value |
|-------|-------|
| Max response body | 10 MB |
| Max redirects | 5 |
| HTTP timeout | 30s |
| Search timeout | 15s |
| Read content | 50,000 chars |
| Search content per result | 5,000 chars |
| Max search results | 10 |
| Max extracted links | 100 |

## Architecture

```
Agent
  | JSON-RPC 2.0 over stdio
  v
Web MCP Server (main.go)
  |
  +---> HTTP Client (pipeline/fetcher.go) ---> SSRF Validator
  |
  +---> Browser Pool (pipeline/browser.go) ---> Headless Chromium
  |
  +---> Readability (pipeline/readability.go) ---> Content extraction
  |
  +---> Markdown (pipeline/markdown.go) ---> HTML conversion
  |
  +---> Metadata (pipeline/metadata.go) ---> OG/Twitter/JSON-LD
```

MCP protocol version `2024-11-05`. Methods: `initialize`, `tools/list`, `tools/call`, `ping`.

## Dependencies

- Go 1.24+
- Chromium/Chrome (auto-detected or downloaded by Rod launcher)
- `go-readability` - Mozilla Readability algorithm
- `html-to-markdown` - HTML to Markdown conversion
- `goquery` - CSS selector parsing
- `go-rod/rod` - Headless browser control
