package langfuse

var fallbackPrompts = map[string]*Prompt{
	"alicia/agent/system-prompt": {
		Name:    "alicia/agent/system-prompt",
		Version: 0,
		Prompt:  "You are Alicia, a helpful AI assistant.",
		Labels:  []string{"fallback"},
	},
	"alicia/agent/continue-response": {
		Name:    "alicia/agent/continue-response",
		Version: 0,
		Prompt:  "Please continue your previous response.",
		Labels:  []string{"fallback"},
	},
	"alicia/agent/conversation-title": {
		Name:    "alicia/agent/conversation-title",
		Version: 0,
		Prompt:  "Generate a short title (under 50 chars) for this conversation.\n\nUser message: {{user_message}}\n{{#assistant_message}}Assistant response: {{assistant_message}}{{/assistant_message}}\n\nRespond with ONLY the title, no quotes or explanation.",
		Labels:  []string{"fallback"},
	},
	"alicia/pareto/seed-default": {
		Name:    "alicia/pareto/seed-default",
		Version: 0,
		Prompt: `You are solving a query task.
1. First understand what information is needed
2. Identify relevant data sources and relationships
3. Construct appropriate queries or tool calls
4. Verify results make sense before concluding
5. Synthesize findings into a clear, accurate answer`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/seed-methodical": {
		Name:    "alicia/pareto/seed-methodical",
		Version: 0,
		Prompt: `Approach this query methodically:
1. Break down the question into sub-questions
2. Address each sub-question with targeted tool use
3. Combine partial answers into a coherent response
4. Double-check for consistency`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/seed-efficiency": {
		Name:    "alicia/pareto/seed-efficiency",
		Version: 0,
		Prompt: `Focus on efficiency:
1. Identify the most direct path to the answer
2. Use minimal tool calls - prefer broader queries over multiple narrow ones
3. Avoid redundant operations
4. Provide a concise, accurate answer`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/seed-accuracy": {
		Name:    "alicia/pareto/seed-accuracy",
		Version: 0,
		Prompt: `Prioritize accuracy:
1. Gather comprehensive information first
2. Cross-reference data from multiple sources when possible
3. Be explicit about uncertainty
4. Prefer verified facts over inferences`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/seed-stepbystep": {
		Name:    "alicia/pareto/seed-stepbystep",
		Version: 0,
		Prompt: `Think step by step:
1. What is the user really asking?
2. What information do I need?
3. What's the best way to get it?
4. How do I present the answer clearly?`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/eval-effectiveness": {
		Name:    "alicia/pareto/eval-effectiveness",
		Version: 0,
		Prompt: `Rate how effectively this response answers the user's question.

QUESTION: {{question}}

RESPONSE: {{response}}

Rate on a 1-5 scale:
1: Complete failure - no answer, error, or completely wrong topic
2: Poor - attempted but missed the point or gave unusable answer
3: Partial - addressed the question but incomplete or partially wrong
4: Good - answered the question with minor issues
5: Excellent - fully and correctly answered the question

Output format: STARS: [1-5]`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/eval-quality": {
		Name:    "alicia/pareto/eval-quality",
		Version: 0,
		Prompt: `Rate the quality of this answer's content and presentation.

QUESTION: {{question}}

ANSWER: {{answer}}

Rate on a 1-5 scale:
1: Terrible - incoherent, unhelpful, or harmful
2: Poor - hard to understand, poorly organized, or mostly unhelpful
3: Acceptable - understandable but could be clearer or more helpful
4: Good - clear, well-organized, and helpful
5: Excellent - exceptionally clear, insightful, and perfectly addresses the need

Output format: STARS: [1-5]`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/eval-hallucination": {
		Name:    "alicia/pareto/eval-hallucination",
		Version: 0,
		Prompt: `Rate the factual accuracy of this answer based on the tool outputs.

TOOL OUTPUTS (the only source of truth):
{{tool_outputs}}

ANSWER:
{{answer}}

Rate on a 1-5 scale:
1: Severe hallucination - makes up facts not in tool outputs, contradicts data
2: Significant hallucination - several unsupported claims or embellishments
3: Some hallucination - a few minor unsupported details
4: Mostly accurate - reasonable inferences, no major fabrications
5: Fully accurate - all claims supported by tool outputs

Output format: STARS: [1-5]`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/eval-specificity": {
		Name:    "alicia/pareto/eval-specificity",
		Version: 0,
		Prompt: `Rate whether the answer's level of detail matches what the question needs.

QUESTION: {{question}}

ANSWER: {{answer}}

Rate on a 1-5 scale:
1: Completely wrong level - way too vague for specific question, or overwhelming detail for simple question
2: Poor match - noticeably too vague or too detailed
3: Acceptable - somewhat appropriate but could be better calibrated
4: Good match - appropriate level of detail for the question type
5: Perfect match - exactly the right amount of detail and depth

Output format: STARS: [1-5]`,
		Labels: []string{"fallback"},
	},
	"alicia/garden/sql-debug-system": {
		Name:    "alicia/garden/sql-debug-system",
		Version: 0,
		Prompt: `You are a SQL debugging assistant for a PostgreSQL database.
Your job is to provide a single, actionable hint to fix SQL errors.

Rules:
- Be concise: one or two sentences max
- Be specific: mention exact column/table names when possible
- Suggest using describe_table to check schema
- If the error is about a missing column, suggest the correct column name if you can infer it from context`,
		Labels: []string{"fallback"},
	},
	"alicia/garden/sql-debug-user": {
		Name:    "alicia/garden/sql-debug-user",
		Version: 0,
		Prompt: `SQL Query:
{{sql}}

Error:
{{error}}

Provide a brief, actionable hint to fix this error.`,
		Labels: []string{"fallback"},
	},
	"alicia/garden/schema-qa-system": {
		Name:    "alicia/garden/schema-qa-system",
		Version: 0,
		Prompt: `You are a database documentation assistant. Answer questions about the database schema based on the provided documentation.

Rules:
- Be accurate and specific
- Reference exact table and column names
- Include example SQL queries when helpful
- If the documentation doesn't contain the answer, say so and suggest using describe_table`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/mutation-strategy": {
		Name:    "alicia/pareto/mutation-strategy",
		Version: 0,
		Prompt: `Analyze this execution trace and improve the strategy.

ORIGINAL QUERY: {{query}}

STRATEGY USED:
{{strategy}}

EXECUTION TRACE:
{{trace}}

FEEDBACK: {{feedback}}

ACCUMULATED LESSONS:
{{lessons}}

Based on what worked and what didn't, provide:
1. LESSONS_LEARNED: New lessons from this attempt (bullet points, each on its own line starting with "- ")
2. IMPROVED_STRATEGY: A better strategy prompt for the next attempt

The improved strategy should be specific, actionable, and address the failures observed.

Format your response exactly like this:
LESSONS_LEARNED:
- lesson 1
- lesson 2

IMPROVED_STRATEGY:
Your improved strategy text here...`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/mutation-crossover": {
		Name:    "alicia/pareto/mutation-crossover",
		Version: 0,
		Prompt: `Merge these two successful strategies into one.

STRATEGY 1 (from path with scores: effectiveness={{scores1}}):
{{strategy1}}

Lessons learned:
{{lessons1}}

STRATEGY 2 (from path with scores: effectiveness={{scores2}}):
{{strategy2}}

Lessons learned:
{{lessons2}}

Create a MERGED_STRATEGY that combines the best elements:
- Keep what makes each strategy effective
- Resolve conflicts in favor of accuracy
- Be specific and actionable

Format your response exactly like this:
MERGED_STRATEGY:
Your merged strategy text here...`,
		Labels: []string{"fallback"},
	},
	"alicia/pareto/thinking-status": {
		Name:    "alicia/pareto/thinking-status",
		Version: 0,
		Prompt: `Generate a short, fun status message (1-10 words) about working on this question.

Question: {{question}}
Current approach: {{strategy}}
Progress: {{progress}}%

Be witty, playful, or encouraging. Match the tone to the question type.

Examples:
- "Crunching the numbers..."
- "Diving deep into data..."
- "Almost got it!"
- "Exploring possibilities..."
- "Connecting the dots..."

Output ONLY the message, nothing else.`,
		Labels: []string{"fallback"},
	},
}

func getFallbackPrompt(name string) (*Prompt, bool) {
	prompt, ok := fallbackPrompts[name]
	return prompt, ok
}
