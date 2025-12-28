package baselines

// ToolSelectionPrompt is the baseline prompt for selecting which tool to use
var ToolSelectionPrompt = `Given the user's intent and available tools, decide which tool (if any) to use.

User Intent: {{.UserIntent}}
Conversation Context: {{.Context}}

Available Tools:
{{range .Tools}}
- {{.Name}}: {{.Description}}
  Parameters: {{.Parameters}}
{{end}}

If a tool is needed, respond with:
TOOL: <tool_name>
ARGUMENTS: <json arguments>
REASONING: <why this tool>

If no tool is needed, respond with:
TOOL: none
REASONING: <why no tool is needed>`
