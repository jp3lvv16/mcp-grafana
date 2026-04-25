// Package tools provides MCP tool implementations for interacting with Grafana.
// Each tool corresponds to a specific Grafana API capability exposed via the
// Model Context Protocol (MCP) server.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GrafanaClient holds configuration for communicating with a Grafana instance.
type GrafanaClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// NewGrafanaClient creates a new GrafanaClient with the given base URL and API key.
func NewGrafanaClient(baseURL, apiKey string) *GrafanaClient {
	return &GrafanaClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTP:    &http.Client{},
	}
}

// doRequest performs an authenticated HTTP GET request to the Grafana API.
func (c *GrafanaClient) doRequest(ctx context.Context, path string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.HTTP.Do(req)
}

// RegisterTools registers all available Grafana MCP tools with the given server.
func RegisterTools(s *server.MCPServer, client *GrafanaClient) {
	registerSearchDashboards(s, client)
	registerGetDashboard(s, client)
	registerListDataSources(s, client)
}

// registerSearchDashboards registers the search_dashboards tool.
func registerSearchDashboards(s *server.MCPServer, client *GrafanaClient) {
	s.AddTool(
		mcp.NewTool("search_dashboards",
			mcp.WithDescription("Search for dashboards in Grafana by query string."),
			mcp.WithString("query",
				mcp.Description("Search query to filter dashboards by title."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, _ := req.Params.Arguments["query"].(string)
			path := "/api/search?type=dash-db"
			if query != "" {
				path += "&query=" + query
			}
			resp, err := client.doRequest(ctx, path)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("search dashboards: %v", err)), nil
			}
			defer resp.Body.Close()
			var result any
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("decode response: %v", err)), nil
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}

// registerGetDashboard registers the get_dashboard tool.
func registerGetDashboard(s *server.MCPServer, client *GrafanaClient) {
	s.AddTool(
		mcp.NewTool("get_dashboard",
			mcp.WithDescription("Retrieve a Grafana dashboard by its UID."),
			mcp.WithString("uid",
				mcp.Required(),
				mcp.Description("The unique identifier (UID) of the dashboard."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			uid, ok := req.Params.Arguments["uid"].(string)
			if !ok || uid == "" {
				return mcp.NewToolResultError("uid is required"), nil
			}
			resp, err := client.doRequest(ctx, "/api/dashboards/uid/"+uid)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("get dashboard: %v", err)), nil
			}
			defer resp.Body.Close()
			var result any
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("decode response: %v", err)), nil
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}

// registerListDataSources registers the list_datasources tool.
func registerListDataSources(s *server.MCPServer, client *GrafanaClient) {
	s.AddTool(
		mcp.NewTool("list_datasources",
			mcp.WithDescription("List all configured data sources in Grafana."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			resp, err := client.doRequest(ctx, "/api/datasources")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list datasources: %v", err)), nil
			}
			defer resp.Body.Close()
			var result any
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("decode response: %v", err)), nil
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}
