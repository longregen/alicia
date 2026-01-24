package main

import "github.com/longregen/alicia/shared/mcp"

func AllTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "get_time",
			Description: "Get the current time and timezone from the user's phone",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_date",
			Description: "Get the current date and day of week from the user's phone",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_battery",
			Description: "Get battery level and charging state of the user's phone",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_location",
			Description: "Get the user's current location (city-level coarse location) from their phone",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "read_screen",
			Description: "Read the text currently displayed on the user's phone screen",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_clipboard",
			Description: "Get the current clipboard contents from the user's phone",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}
