package baselines

// MemoryExtractionPrompt is the baseline prompt for extracting memories from conversations
var MemoryExtractionPrompt = `Extract key facts from this conversation that should be remembered for future interactions.

Conversation:
{{.Conversation}}

Extract facts that are:
- Personal preferences (e.g., "prefers TypeScript", "works on React projects")
- Project context (e.g., "building an e-commerce app", "using PostgreSQL")
- Instructions (e.g., "always use async/await", "prefers concise responses")

Format each fact as a single sentence. Rate importance 1-5 (5 = always remember).

FACTS:
1. [fact] (importance: X)
2. [fact] (importance: X)
...`

// ToolResultFormatterPrompt is the baseline prompt for formatting tool results
var ToolResultFormatterPrompt = `Format this tool result for optimal LLM consumption.

Tool: {{.ToolName}}
Raw Result: {{.RawResult}}
User Context: {{.UserContext}}

Guidelines:
- Extract the most relevant information for the user's query
- Summarize verbose output while preserving key details
- Highlight actionable items
- Format for readability

Formatted Result:`
