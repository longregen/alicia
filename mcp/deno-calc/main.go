package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/config"
	"github.com/longregen/alicia/shared/mcp"
)

func main() {
	// Initialize OpenTelemetry (provides tee'd logger: stderr JSON + OTLP export)
	otelEndpoint := config.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://alicia-data.hjkl.lol")
	result, err := otel.Init(otel.Config{
		ServiceName:  "mcp-deno-calc",
		Environment:  config.GetEnv("ENVIRONMENT", ""),
		OTLPEndpoint: otelEndpoint,
	})
	if err != nil {
		// Fallback to plain stderr logger if OTel fails
		slog.SetDefault(slog.New(otel.NewPrettyHandler()))
		slog.Warn("otel init failed, continuing without export", "error", err)
	} else {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			result.Shutdown(shutdownCtx)
		}()
		slog.SetDefault(result.Logger)
		slog.Info("otel initialized", "endpoint", otelEndpoint)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req mcp.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		resp := handleRequest(ctx, req)
		if resp != nil {
			data, _ := json.Marshal(resp)
			fmt.Println(string(data))
		}
	}
}

func handleRequest(ctx context.Context, req mcp.Request) *mcp.Response {
	switch req.Method {
	case "initialize":
		return mcp.NewResponse(req.ID, mcp.InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: mcp.ServerCapabilities{
				Tools: &mcp.ToolsCapability{},
			},
			ServerInfo: mcp.ServerInfo{
				Name:    "mcp-deno-calc",
				Version: "1.0.0",
			},
		})

	case "initialized":
		return nil // notification, no response

	case "tools/list":
		return mcp.NewResponse(req.ID, mcp.ToolsListResult{
			Tools: []mcp.Tool{
				{
					Name:        "calculate",
					Description: "Execute JavaScript/TypeScript code using Deno for calculations. Runs completely offline with no network access. Returns the result of the last expression. Use for math, data transformations, string manipulation, etc.",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"code": map[string]any{
								"type":        "string",
								"description": "JavaScript/TypeScript code to execute. The result of the last expression will be returned. Use console.log() for additional output.",
							},
						},
						"required": []string{"code"},
					},
				},
			},
		})

	case "tools/call":
		params, err := mcp.DecodeParams(req.Params)
		if err != nil {
			return mcp.NewErrorResponse(req.ID, mcp.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}

		// Start span with trace context from _meta
		ctx, span := otel.StartMCPToolSpan(ctx, "mcp-deno-calc", "deno-calc", params.Name, params.Meta)
		defer span.End()

		if params.Name != "calculate" {
			otel.RecordToolError(span, fmt.Errorf("unknown tool: %s", params.Name))
			return mcp.NewErrorResponse(req.ID, mcp.ErrCodeInvalidParams, fmt.Sprintf("Unknown tool: %s", params.Name))
		}

		code, ok := params.Arguments["code"].(string)
		if !ok {
			otel.RecordToolError(span, fmt.Errorf("missing or invalid 'code' argument"))
			return mcp.NewErrorResponse(req.ID, mcp.ErrCodeInvalidParams, "Missing or invalid 'code' argument")
		}

		result, err := executeCode(ctx, code)
		if err != nil {
			span.RecordError(err)
			otel.EndMCPToolSpan(span, true, 0)
			return mcp.NewResponse(req.ID, mcp.NewToolError(fmt.Sprintf("Error: %v", err)))
		}

		otel.EndMCPToolSpan(span, false, len(result))
		return mcp.NewResponse(req.ID, mcp.NewToolResult(result))

	default:
		return mcp.NewErrorResponse(req.ID, mcp.ErrCodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func executeCode(ctx context.Context, code string) (string, error) {
	// Wrap code to capture last expression result
	wrappedCode := fmt.Sprintf(`
const __result = await (async () => {
%s
})();
if (__result !== undefined) {
	console.log(typeof __result === 'object' ? JSON.stringify(__result, null, 2) : __result);
}
`, code)

	// Create a temporary file for the code
	tmpFile, err := os.CreateTemp("", "deno-calc-*.ts")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(wrappedCode); err != nil {
		return "", fmt.Errorf("failed to write code: %w", err)
	}
	tmpFile.Close()

	// Run with Deno - completely offline, no permissions
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "deno", "run",
		"--no-remote",      // No network access
		"--no-npm",         // No npm packages
		"--no-config",      // Ignore config files
		"--allow-read="+tmpFile.Name(), // Only allow reading the temp file
		tmpFile.Name(),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("execution timed out (30s limit)")
		}
		// Include output in error message for debugging
		return "", fmt.Errorf("%s\n%s", err, strings.TrimSpace(string(output)))
	}

	return strings.TrimSpace(string(output)), nil
}
