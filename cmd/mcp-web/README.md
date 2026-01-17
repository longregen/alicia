# MCP Web Browser

A Model Context Protocol (MCP) server that provides web browsing capabilities for LLM agents. Converts web content to clean, LLM-friendly markdown format.

## Features

- **Content Extraction**: Automatically extracts main article content, removes ads/navigation/boilerplate
- **HTML to Markdown**: Converts web pages to clean markdown for token-efficient LLM consumption
- **Metadata Extraction**: Extracts Open Graph, Twitter Card, JSON-LD structured data
- **Web Search**: Search the web via DuckDuckGo with optional content fetching
- **SSRF Protection**: Built-in security against server-side request forgery attacks

## Tools

| Tool | Description |
|------|-------------|
| `read` | Fetches a URL and returns clean markdown content (best for articles/docs) |
| `fetch_raw` | Returns raw HTTP response with headers (best for APIs) |
| `fetch_structured` | Extracts data using CSS-like selectors |
| `search` | Web search with optional content fetching |
| `extract_links` | Extracts all links from a page with filtering |
| `extract_metadata` | Extracts page metadata (OG, Twitter, JSON-LD) |
| `screenshot` | Captures page screenshots using headless Chromium |

## Installation

```bash
# Build from source
CGO_ENABLED=0 go build -o mcp-web ./cmd/mcp-web

# Or install directly
go install github.com/longregen/alicia/cmd/mcp-web@latest
```

## Usage

### As stdio MCP Server

```bash
./mcp-web
```

The server communicates via JSON-RPC 2.0 over stdin/stdout.

### With Claude Desktop

Add to your Claude Desktop config (`~/.config/claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "web": {
      "command": "/path/to/mcp-web"
    }
  }
}
```

### With Alicia

Register as an MCP server in Alicia's configuration or via the API.

## Tool Examples

### read

Fetches and converts a web page to markdown. Supports JavaScript rendering for SPAs and dynamic content:

```json
{
  "name": "read",
  "arguments": {
    "url": "https://example.com/article",
    "include_links": true,
    "max_length": 50000,
    "wait_for": "networkidle",
    "wait_ms": 1000
  }
}
```

Response:
```json
{
  "url": "https://example.com/article",
  "title": "Article Title",
  "content": "# Article Title\n\nMain content in markdown...",
  "word_count": 1234,
  "estimated_tokens": 890,
  "js_rendered": true
}
```

### fetch_raw

Fetches raw HTTP response (useful for APIs):

```json
{
  "name": "fetch_raw",
  "arguments": {
    "url": "https://api.example.com/data",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer token123"
    }
  }
}
```

### search

Web search with optional content fetching:

```json
{
  "name": "search",
  "arguments": {
    "query": "rust programming language",
    "num_results": 5,
    "fetch_content": true
  }
}
```

### extract_links

Extract links from a page:

```json
{
  "name": "extract_links",
  "arguments": {
    "url": "https://docs.example.com",
    "filter": "internal",
    "pattern": "/api/"
  }
}
```

### extract_metadata

Extract page metadata:

```json
{
  "name": "extract_metadata",
  "arguments": {
    "url": "https://example.com/article"
  }
}
```

### fetch_structured

Extract specific elements using selectors:

```json
{
  "name": "fetch_structured",
  "arguments": {
    "url": "https://example.com/product",
    "selectors": {
      "title": "h1",
      "price": ".price",
      "links": "a[href]"
    }
  }
}
```

## Security

The server includes built-in SSRF protection:

- Blocks requests to private/internal IP ranges
- Blocks requests to metadata endpoints (AWS, GCP, Azure)
- Validates URL schemes (only HTTP/HTTPS allowed)
- Optional allowlist for permitted hosts

To configure an allowlist, set `security.AllowedHosts` before starting the server.

## Architecture

```
cmd/mcp-web/
├── main.go           # MCP server entry point
├── types.go          # JSON-RPC protocol types
├── pipeline/
│   ├── browser.go    # Headless browser pool (rod)
│   ├── fetcher.go    # HTTP fetching with SSRF protection
│   ├── readability.go # Main content extraction
│   ├── markdown.go   # HTML to Markdown conversion
│   └── metadata.go   # Metadata extraction
├── security/
│   └── validator.go  # URL validation
└── tools/
    ├── registry.go   # Tool registration
    ├── read.go       # read tool (with JS rendering support)
    ├── fetch.go      # fetch_raw, fetch_structured tools
    ├── search.go     # search tool
    ├── links.go      # extract_links tool
    ├── metadata.go   # extract_metadata tool
    └── screenshot.go # screenshot tool
```

## Dependencies

- `codeberg.org/readeck/go-readability/v2` - Mozilla Readability content extraction
- `github.com/JohannesKaufmann/html-to-markdown/v2` - HTML to Markdown conversion
- `github.com/PuerkitoBio/goquery` - CSS selector-based HTML parsing
- `github.com/go-rod/rod` - Headless browser for screenshots and JS rendering
- `golang.org/x/net/html` - HTML parsing

## License

Same as Alicia project.
