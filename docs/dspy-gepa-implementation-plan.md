# DSPy + GEPA Implementation Plan for Alicia

## Executive Summary

This document outlines a plan to integrate DSPy-like prompt programming and GEPA optimization into Alicia. Rather than reimplementing from scratch, we recommend leveraging existing Go implementations (particularly `dspy-go`) and focusing on Alicia-specific integration patterns.

## Background

### What is DSPy?

DSPy (Declarative Self-improving Python) is Stanford NLP's framework for **programming language models** rather than prompting them. Key concepts:

1. **Signatures**: Declarative input/output specifications
   ```python
   "question -> answer"
   "context, question -> reasoning, answer"
   ```

2. **Modules**: Composable LLM building blocks
   - `Predict` - Basic LLM call
   - `ChainOfThought` - Adds reasoning steps
   - `ReAct` - Tool-using agent
   - Custom modules via composition

3. **Optimizers**: Automatic prompt improvement algorithms
   - `LabeledFewShot` - Use labeled examples
   - `BootstrapFewShot` - Generate demonstrations
   - `MIPROv2` - Joint instruction + example optimization
   - `GEPA` - State-of-the-art reflective evolution

### What is GEPA?

GEPA (Genetic-Pareto) is DSPy's most advanced optimizer:

- **Reflective Mutation**: Uses a strong LLM to analyze failures and propose improved instructions
- **Pareto Frontier**: Maintains diverse candidate pool (generalists + specialists)
- **Coverage-based Selection**: Samples candidates proportional to instances they solve best
- **Performance**: Outperforms RL by 20%, uses 35x fewer evaluations

**Algorithm Flow:**
```
1. Initialize with unoptimized program
2. Sample candidate from Pareto frontier (prob ∝ coverage)
3. Run on minibatch, collect traces + feedback
4. Reflect and propose new instruction
5. If improved, evaluate on full validation set
6. Update Pareto frontier
7. Repeat until budget exhausted
```

## dspy-go Library Status

**Repository**: https://github.com/XiaoConstantine/dspy-go

| Metric | Value |
|--------|-------|
| Stars | 140 |
| Forks | 9 |
| Commits | 411 |
| License | MIT |

### Implemented Features

- ✅ **GEPA optimizer** - State-of-the-art reflective evolution
- ✅ **MIPRO optimizer** - Joint instruction + example optimization
- ✅ **SIMBA optimizer** - Additional optimization strategy
- ✅ **BootstrapFewShot** - Generate demonstrations
- ✅ **Core modules** - Predict, ChainOfThought, ReAct, MultiChainComparison, Refine, Parallel
- ✅ **Multi-provider LLM** - Anthropic Claude, Google Gemini, OpenAI, Ollama
- ✅ **Structured outputs** - Via `.WithStructuredOutput()`
- ✅ **Multimodal** - Vision support
- ✅ **Tool integration** - MCP server support
- ✅ **Dataset management** - GSM8K, HotPotQA
- ✅ **CLI tool** - Interactive optimizer testing

The library appears actively maintained with comprehensive documentation.

## Initial Prompts (Baseline)

Before GEPA optimization, we need hand-written baseline prompts that GEPA will improve upon. These are the starting points:

### Conversation Response Prompt

```go
// internal/prompt/baselines/conversation.go
package baselines

var ConversationResponsePrompt = `You are Alicia, a helpful AI assistant with memory capabilities.

You have access to:
- Conversation context from this session
- Relevant memories from past interactions
- Tools for file operations, web search, and more

Guidelines:
1. Be conversational and natural in your responses
2. Reference relevant memories when they add value
3. Use tools when the user's request requires external information
4. Be concise but thorough

Context: {{.Context}}
Memories: {{.Memories}}
User Message: {{.UserMessage}}

Respond helpfully to the user's message.`
```

### Tool Selection Prompt

```go
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
```

### Memory Extraction Prompt

```go
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
```

### Tool Result Formatting Prompt

```go
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
```

These baseline prompts will be optimized by GEPA to improve:
- Response quality and helpfulness
- Tool selection accuracy
- Memory extraction precision
- Result formatting clarity

---

## Implementation Strategy

### Phase 1: Core Integration

#### 1.1 Add dspy-go Dependency

```bash
go get github.com/XiaoConstantine/dspy-go
```

#### 1.2 Create Prompt Abstraction Layer

New package: `internal/prompt/`

```go
// internal/prompt/signature.go
package prompt

import (
    "github.com/XiaoConstantine/dspy-go/core"
)

// Signature wraps dspy-go's signature with Alicia-specific features
type Signature struct {
    core.Signature
    Name        string
    Description string
    Version     int
}

// PredefinedSignatures for common Alicia use cases
var (
    ConversationResponse = MustParseSignature(
        "context, user_message, memories -> response",
    )

    ToolSelection = MustParseSignature(
        "user_intent, available_tools -> tool_name, reasoning",
    )

    MemoryExtraction = MustParseSignature(
        "conversation -> key_facts: list[str], importance: int",
    )
)
```

#### 1.3 Create Module Wrappers

```go
// internal/prompt/modules.go
package prompt

import (
    "context"
    "github.com/XiaoConstantine/dspy-go/modules"
)

// AliciaPredict wraps dspy-go Predict with Alicia integration
type AliciaPredict struct {
    *modules.Predict
    tracer   Tracer
    metrics  MetricsCollector
}

func NewAliciaPredict(sig Signature, opts ...Option) *AliciaPredict {
    return &AliciaPredict{
        Predict: modules.NewPredict(sig.Signature),
    }
}

func (p *AliciaPredict) Forward(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    // Pre-execution hooks
    span := p.tracer.StartSpan(ctx, "predict")
    defer span.End()

    // Execute
    outputs, err := p.Predict.Forward(ctx, inputs)

    // Post-execution metrics
    p.metrics.RecordExecution(span, inputs, outputs, err)

    return outputs, err
}
```

#### 1.4 Integrate with LLM Service

```go
// internal/prompt/llm_adapter.go
package prompt

import (
    "context"
    "github.com/XiaoConstantine/dspy-go/llms"
    "alicia/internal/ports"
)

// LLMServiceAdapter adapts Alicia's LLMService to dspy-go's LLM interface
type LLMServiceAdapter struct {
    service ports.LLMService
}

func NewLLMServiceAdapter(service ports.LLMService) *LLMServiceAdapter {
    return &LLMServiceAdapter{service: service}
}

func (a *LLMServiceAdapter) Generate(ctx context.Context, prompt string, opts ...llms.Option) (string, error) {
    messages := []ports.LLMMessage{
        {Role: "user", Content: prompt},
    }

    resp, err := a.service.Chat(ctx, messages)
    if err != nil {
        return "", err
    }

    return resp.Content, nil
}
```

### Phase 2: GEPA Optimization Service

#### 2.1 Define Optimization Repository

```go
// internal/ports/repositories.go (additions)

type PromptOptimizationRepository interface {
    // Optimization runs
    CreateRun(ctx context.Context, run *OptimizationRun) error
    GetRun(ctx context.Context, id string) (*OptimizationRun, error)
    UpdateRun(ctx context.Context, run *OptimizationRun) error

    // Candidates
    SaveCandidate(ctx context.Context, runID string, candidate *PromptCandidate) error
    GetCandidates(ctx context.Context, runID string) ([]*PromptCandidate, error)
    GetBestCandidate(ctx context.Context, runID string) (*PromptCandidate, error)

    // Evaluations
    SaveEvaluation(ctx context.Context, eval *PromptEvaluation) error
    GetEvaluations(ctx context.Context, candidateID string) ([]*PromptEvaluation, error)
}

type OptimizationRun struct {
    ID            string
    SignatureName string
    Status        OptimizationStatus
    Config        GEPAConfig
    BestScore     float64
    Iterations    int
    CreatedAt     time.Time
    CompletedAt   *time.Time
}

type PromptCandidate struct {
    ID           string
    RunID        string
    Instructions string
    Demos        []Example
    Coverage     int
    AvgScore     float64
    Generation   int
    ParentID     *string
    CreatedAt    time.Time
}

type PromptEvaluation struct {
    ID          string
    CandidateID string
    ExampleID   string
    Score       float64
    Feedback    string
    Trace       json.RawMessage
    CreatedAt   time.Time
}
```

#### 2.2 Create Optimization Service

```go
// internal/application/services/optimization.go
package services

import (
    "context"
    "github.com/XiaoConstantine/dspy-go/optimizers"
    "alicia/internal/ports"
    "alicia/internal/prompt"
)

type OptimizationService struct {
    repo         ports.PromptOptimizationRepository
    llmService   ports.LLMService
    reflectionLM ports.LLMService // Strong model for GEPA reflection
}

func NewOptimizationService(
    repo ports.PromptOptimizationRepository,
    llmService ports.LLMService,
    reflectionLM ports.LLMService,
) *OptimizationService {
    return &OptimizationService{
        repo:         repo,
        llmService:   llmService,
        reflectionLM: reflectionLM,
    }
}

// OptimizeSignature runs GEPA optimization on a signature
func (s *OptimizationService) OptimizeSignature(
    ctx context.Context,
    sig prompt.Signature,
    trainset []prompt.Example,
    valset []prompt.Example,
    metric prompt.Metric,
    config GEPAConfig,
) (*OptimizedProgram, error) {
    // Create optimization run record
    run := &ports.OptimizationRun{
        ID:            generateID(),
        SignatureName: sig.Name,
        Status:        StatusRunning,
        Config:        config,
    }
    if err := s.repo.CreateRun(ctx, run); err != nil {
        return nil, err
    }

    // Create dspy-go optimizer
    gepa := optimizers.NewGEPA(
        metric.ToGEPAMetric(),
        optimizers.WithAuto(config.Budget),
        optimizers.WithNumThreads(config.NumThreads),
        optimizers.WithReflectionLM(prompt.NewLLMServiceAdapter(s.reflectionLM)),
    )

    // Create student program
    student := prompt.NewAliciaPredict(sig)

    // Run optimization with progress tracking
    optimized, err := gepa.Compile(
        ctx,
        student,
        trainset,
        valset,
        optimizers.WithProgressCallback(func(progress Progress) {
            s.recordProgress(ctx, run.ID, progress)
        }),
    )
    if err != nil {
        run.Status = StatusFailed
        s.repo.UpdateRun(ctx, run)
        return nil, err
    }

    // Save result
    run.Status = StatusCompleted
    run.BestScore = optimized.BestScore()
    run.CompletedAt = timePtr(time.Now())
    s.repo.UpdateRun(ctx, run)

    return optimized, nil
}
```

### Phase 3: Tool Usage Optimization

GEPA has built-in support for tool optimization via `enable_tool_optimization`. This phase focuses on optimizing the entire tool lifecycle.

#### 3.1 Tool Optimization Targets

```
┌─────────────────────────────────────────────────────────────────┐
│                    Tool Usage Optimization                       │
├─────────────────────────────────────────────────────────────────┤
│  1. Tool Descriptions    → Better LLM understanding of when     │
│  2. Tool Arguments       → Improved argument generation         │
│  3. Tool Selection       → Smarter tool choice decisions        │
│  4. Result Formatting    → Optimized output for LLM consumption │
│  5. Result Summarization → Condensing verbose tool outputs      │
│  6. Error Recovery       → Better handling of tool failures     │
└─────────────────────────────────────────────────────────────────┘
```

#### 3.2 Optimizable Tool Definition

```go
// internal/prompt/tool_optimization.go
package prompt

import (
    "context"
    "alicia/internal/domain/models"
)

// OptimizableTool wraps a tool with optimizable components
type OptimizableTool struct {
    BaseTool       *models.Tool

    // Optimizable fields
    Description    string            // Optimized description for LLM
    Schema         map[string]any    // Optimized parameter schema
    ResultTemplate string            // Template for formatting results
    Examples       []ToolExample     // Few-shot examples

    // Metadata
    Version        int
    OptimizedAt    time.Time
}

// ToolExample demonstrates correct tool usage
type ToolExample struct {
    UserIntent   string         // What the user wanted
    Arguments    map[string]any // Correct arguments
    RawResult    any            // Raw tool output
    FormattedResult string      // Optimized formatted result
    Explanation  string         // Why this was the right choice
}

// ToolOptimizationSignature defines what we're optimizing
var ToolDescriptionSignature = MustParseSignature(
    "tool_name, current_description, usage_examples, failure_cases -> optimized_description",
)

var ToolResultFormatterSignature = MustParseSignature(
    "tool_name, raw_result, user_context -> formatted_result",
)

var ToolSelectionSignature = MustParseSignature(
    "user_intent, conversation_context, available_tools: list[Tool] -> selected_tool, arguments, reasoning",
)
```

#### 3.3 Tool Description Optimizer

```go
// internal/application/services/tool_optimization.go
package services

import (
    "context"
    "alicia/internal/prompt"
    "alicia/internal/ports"
)

type ToolOptimizationService struct {
    toolService     ports.ToolService
    toolUseRepo     ports.ToolUseRepository
    optService      *OptimizationService
    llmService      ports.LLMService
}

// OptimizeToolDescriptions improves tool descriptions based on usage patterns
func (s *ToolOptimizationService) OptimizeToolDescriptions(
    ctx context.Context,
    config GEPAConfig,
) error {
    tools, err := s.toolService.ListEnabled(ctx)
    if err != nil {
        return err
    }

    for _, tool := range tools {
        // Gather usage data
        usageData, err := s.gatherToolUsageData(ctx, tool.Name)
        if err != nil {
            continue
        }

        // Create training examples from successful uses
        trainset := s.createDescriptionTrainset(usageData)
        valset := s.createDescriptionValset(usageData)

        // Optimize description
        optimized, err := s.optService.OptimizeSignature(
            ctx,
            prompt.ToolDescriptionSignature,
            trainset,
            valset,
            &ToolDescriptionMetric{},
            config,
        )
        if err != nil {
            continue
        }

        // Update tool with optimized description
        newDescription := optimized.GetOutput("optimized_description")
        s.toolService.UpdateDescription(ctx, tool.ID, newDescription)
    }

    return nil
}

// gatherToolUsageData collects historical tool usage for analysis
func (s *ToolOptimizationService) gatherToolUsageData(
    ctx context.Context,
    toolName string,
) (*ToolUsageData, error) {
    uses, err := s.toolUseRepo.GetByToolName(ctx, toolName, 1000)
    if err != nil {
        return nil, err
    }

    return &ToolUsageData{
        ToolName:      toolName,
        SuccessfulUses: filterSuccessful(uses),
        FailedUses:     filterFailed(uses),
        CommonPatterns: extractPatterns(uses),
    }, nil
}
```

#### 3.4 Tool Result Formatter Optimization

```go
// internal/prompt/tool_result_formatter.go
package prompt

import (
    "context"
    "encoding/json"
)

// OptimizedResultFormatter formats tool results for optimal LLM consumption
type OptimizedResultFormatter struct {
    formatters map[string]*ToolFormatter // per-tool formatters
    defaultFormatter *ToolFormatter
}

type ToolFormatter struct {
    Template       string           // Go template for formatting
    MaxLength      int              // Truncation limit
    SummarizeAt    int              // When to trigger summarization
    SummaryPrompt  string           // Prompt for summarization
    KeyFields      []string         // Important fields to preserve
}

// Format applies optimized formatting to tool results
func (f *OptimizedResultFormatter) Format(
    ctx context.Context,
    toolName string,
    result any,
    userContext string,
) (string, error) {
    formatter, ok := f.formatters[toolName]
    if !ok {
        formatter = f.defaultFormatter
    }

    // Convert result to structured format
    resultJSON, err := json.Marshal(result)
    if err != nil {
        return fmt.Sprintf("%v", result), nil
    }

    // Check if summarization is needed
    if len(resultJSON) > formatter.SummarizeAt {
        return f.summarize(ctx, toolName, result, userContext, formatter)
    }

    // Apply template formatting
    return f.applyTemplate(formatter.Template, result)
}

// summarize uses LLM to condense verbose tool output
func (f *OptimizedResultFormatter) summarize(
    ctx context.Context,
    toolName string,
    result any,
    userContext string,
    formatter *ToolFormatter,
) (string, error) {
    // Extract key fields first
    keyData := extractKeyFields(result, formatter.KeyFields)

    // Summarize remaining content
    prompt := fmt.Sprintf(formatter.SummaryPrompt, toolName, result, userContext)
    summary, err := f.llm.Generate(ctx, prompt)
    if err != nil {
        return fmt.Sprintf("%v", keyData), nil
    }

    return fmt.Sprintf("Key data: %v\nSummary: %s", keyData, summary), nil
}
```

#### 3.5 Tool Selection Module with GEPA

```go
// internal/prompt/tool_selection.go
package prompt

import (
    "context"
    "github.com/XiaoConstantine/dspy-go/modules"
)

// ToolSelectionModule decides which tool to use and with what arguments
type ToolSelectionModule struct {
    *modules.ChainOfThought
    availableTools []*models.Tool
}

func NewToolSelectionModule(tools []*models.Tool) *ToolSelectionModule {
    sig := MustParseSignature(`
        user_intent: str,
        conversation_context: str,
        available_tools: list[ToolInfo]
        ->
        should_use_tool: bool,
        selected_tool: str,
        arguments: dict,
        reasoning: str
    `)

    return &ToolSelectionModule{
        ChainOfThought: modules.NewChainOfThought(sig),
        availableTools: tools,
    }
}

func (m *ToolSelectionModule) Forward(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    // Inject tool information
    inputs["available_tools"] = m.formatToolsForLLM()

    // Use ChainOfThought for reasoning
    outputs, err := m.ChainOfThought.Forward(ctx, inputs)
    if err != nil {
        return nil, err
    }

    // Validate selected tool exists
    if selectedTool, ok := outputs["selected_tool"].(string); ok {
        if !m.toolExists(selectedTool) {
            outputs["should_use_tool"] = false
            outputs["reasoning"] = "Selected tool not available"
        }
    }

    return outputs, nil
}

func (m *ToolSelectionModule) formatToolsForLLM() []map[string]any {
    var tools []map[string]any
    for _, t := range m.availableTools {
        tools = append(tools, map[string]any{
            "name":        t.Name,
            "description": t.Description,
            "parameters":  t.Schema,
        })
    }
    return tools
}
```

#### 3.6 Tool Usage Metrics

```go
// internal/prompt/tool_metrics.go
package prompt

// ToolDescriptionMetric evaluates if optimized descriptions improve tool selection
type ToolDescriptionMetric struct {
    llmService ports.LLMService
}

func (m *ToolDescriptionMetric) Score(
    ctx context.Context,
    gold, pred Example,
    trace *Trace,
) (ScoreWithFeedback, error) {
    // Evaluate: Does the LLM correctly understand when to use this tool?
    testCases := gold.Inputs["test_cases"].([]ToolTestCase)

    var correct int
    var feedback strings.Builder

    for _, tc := range testCases {
        // Ask LLM if it would use this tool for the given intent
        wouldUse := m.askLLMAboutToolUsage(ctx, pred.Outputs["description"].(string), tc.UserIntent)

        if wouldUse == tc.ShouldUseTool {
            correct++
        } else {
            feedback.WriteString(fmt.Sprintf(
                "- Intent '%s': expected %v, got %v\n",
                tc.UserIntent, tc.ShouldUseTool, wouldUse,
            ))
        }
    }

    score := float64(correct) / float64(len(testCases))
    return ScoreWithFeedback{Score: score, Feedback: feedback.String()}, nil
}

// ToolResultMetric evaluates if formatted results lead to better responses
type ToolResultMetric struct {
    llmService ports.LLMService
}

func (m *ToolResultMetric) Score(
    ctx context.Context,
    gold, pred Example,
    trace *Trace,
) (ScoreWithFeedback, error) {
    formattedResult := pred.Outputs["formatted_result"].(string)
    userIntent := gold.Inputs["user_intent"].(string)
    expectedAnswer := gold.Outputs["expected_answer"].(string)

    // Test: Can LLM produce correct answer from formatted result?
    testPrompt := fmt.Sprintf(`
Based on this tool result, answer the user's question.

User Question: %s
Tool Result: %s

Provide a direct answer.`,
        userIntent, formattedResult,
    )

    response, err := m.llmService.Chat(ctx, []ports.LLMMessage{
        {Role: "user", Content: testPrompt},
    })
    if err != nil {
        return ScoreWithFeedback{Score: 0, Feedback: err.Error()}, nil
    }

    // Compare response quality
    similarity := semanticSimilarity(response.Content, expectedAnswer)
    feedback := fmt.Sprintf(
        "Expected: %s\nGot: %s\nSimilarity: %.2f",
        expectedAnswer, response.Content, similarity,
    )

    return ScoreWithFeedback{Score: similarity, Feedback: feedback}, nil
}

// ToolArgumentMetric evaluates argument generation quality
type ToolArgumentMetric struct{}

func (m *ToolArgumentMetric) Score(
    ctx context.Context,
    gold, pred Example,
    trace *Trace,
) (ScoreWithFeedback, error) {
    expectedArgs := gold.Outputs["arguments"].(map[string]any)
    actualArgs := pred.Outputs["arguments"].(map[string]any)

    // Check required fields
    schema := gold.Inputs["schema"].(map[string]any)
    required := schema["required"].([]string)

    var missingFields []string
    var wrongValues []string

    for _, field := range required {
        if _, ok := actualArgs[field]; !ok {
            missingFields = append(missingFields, field)
        }
    }

    // Check value correctness
    for key, expectedVal := range expectedArgs {
        if actualVal, ok := actualArgs[key]; ok {
            if !valuesEqual(expectedVal, actualVal) {
                wrongValues = append(wrongValues, fmt.Sprintf(
                    "%s: expected %v, got %v", key, expectedVal, actualVal,
                ))
            }
        }
    }

    // Calculate score
    totalFields := len(expectedArgs)
    correctFields := totalFields - len(missingFields) - len(wrongValues)
    score := float64(correctFields) / float64(totalFields)

    feedback := fmt.Sprintf(
        "Missing: %v\nWrong values: %v",
        missingFields, wrongValues,
    )

    return ScoreWithFeedback{Score: score, Feedback: feedback}, nil
}
```

#### 3.7 Integrated Tool Optimization Pipeline

```go
// internal/application/services/tool_optimization_pipeline.go
package services

// ToolOptimizationPipeline runs comprehensive tool optimization
type ToolOptimizationPipeline struct {
    toolOptService  *ToolOptimizationService
    optService      *OptimizationService
    toolService     ports.ToolService
}

// RunFullOptimization optimizes all aspects of tool usage
func (p *ToolOptimizationPipeline) RunFullOptimization(
    ctx context.Context,
    config GEPAConfig,
) (*ToolOptimizationReport, error) {
    report := &ToolOptimizationReport{
        StartedAt: time.Now(),
    }

    // 1. Optimize tool descriptions
    descResults, err := p.optimizeDescriptions(ctx, config)
    report.DescriptionResults = descResults

    // 2. Optimize result formatters
    formatResults, err := p.optimizeResultFormatters(ctx, config)
    report.FormatterResults = formatResults

    // 3. Optimize tool selection module
    selectionResults, err := p.optimizeToolSelection(ctx, config)
    report.SelectionResults = selectionResults

    // 4. Optimize argument generation
    argResults, err := p.optimizeArgumentGeneration(ctx, config)
    report.ArgumentResults = argResults

    report.CompletedAt = time.Now()
    return report, nil
}

// optimizeToolSelection creates an optimized tool selection module
func (p *ToolOptimizationPipeline) optimizeToolSelection(
    ctx context.Context,
    config GEPAConfig,
) (*SelectionOptimizationResult, error) {
    tools, _ := p.toolService.ListEnabled(ctx)

    // Create training data from historical tool uses
    trainset := p.createToolSelectionTrainset(ctx)
    valset := p.createToolSelectionValset(ctx)

    // Create module
    module := prompt.NewToolSelectionModule(tools)

    // Optimize with GEPA - enable tool optimization flag
    gepa := optimizers.NewGEPA(
        &prompt.ToolSelectionMetric{},
        optimizers.WithAuto(config.Budget),
        optimizers.WithEnableToolOptimization(true), // Key flag!
    )

    optimized, err := gepa.Compile(ctx, module, trainset, valset)
    if err != nil {
        return nil, err
    }

    return &SelectionOptimizationResult{
        OriginalAccuracy:   evalAccuracy(module, valset),
        OptimizedAccuracy:  evalAccuracy(optimized, valset),
        OptimizedModule:    optimized,
    }, nil
}
```

#### 3.8 Database Schema for Tool Optimization

```sql
-- Optimized tool configurations
CREATE TABLE optimized_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_id UUID NOT NULL REFERENCES alicia_tools(id),
    optimized_description TEXT NOT NULL,
    optimized_schema JSONB,
    result_template TEXT,
    examples JSONB,
    version INT NOT NULL DEFAULT 1,
    score FLOAT,
    optimized_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    active BOOLEAN DEFAULT false
);

CREATE INDEX idx_optimized_tools_active ON optimized_tools(tool_id, active);

-- Tool result formatting rules (learned)
CREATE TABLE tool_result_formatters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_name VARCHAR(255) NOT NULL,
    template TEXT NOT NULL,
    max_length INT DEFAULT 2000,
    summarize_at INT DEFAULT 1000,
    summary_prompt TEXT,
    key_fields JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_formatters_tool ON tool_result_formatters(tool_name);

-- Tool usage patterns (for learning)
CREATE TABLE tool_usage_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_name VARCHAR(255) NOT NULL,
    user_intent_pattern TEXT NOT NULL,
    success_rate FLOAT,
    avg_result_quality FLOAT,
    common_arguments JSONB,
    sample_count INT DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_patterns_tool ON tool_usage_patterns(tool_name);
```

### Phase 4: Memory-Aware Optimization

Leverage Alicia's existing memory system for in-context learning:

```go
// internal/prompt/memory_integration.go
package prompt

import (
    "context"
    "alicia/internal/ports"
)

// MemoryAwareModule injects relevant memories as demonstrations
type MemoryAwareModule struct {
    base          Module
    memoryService ports.MemoryService
    embedService  ports.EmbeddingService
    maxDemos      int
}

func NewMemoryAwareModule(
    base Module,
    memoryService ports.MemoryService,
    embedService ports.EmbeddingService,
) *MemoryAwareModule {
    return &MemoryAwareModule{
        base:          base,
        memoryService: memoryService,
        embedService:  embedService,
        maxDemos:      5,
    }
}

func (m *MemoryAwareModule) Forward(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    // Search for relevant memories as potential demonstrations
    query := extractQuery(inputs)
    memories, err := m.memoryService.SearchMemoriesWithRelevance(ctx, query, m.maxDemos)
    if err != nil {
        // Continue without memories on error
        return m.base.Forward(ctx, inputs)
    }

    // Convert memories to demonstrations
    demos := memoriesToDemos(memories)

    // Inject demonstrations into module
    enrichedInputs := inputs
    enrichedInputs["_demonstrations"] = demos

    return m.base.Forward(ctx, enrichedInputs)
}
```

### Phase 5: Evaluation & Metrics

```go
// internal/prompt/metrics.go
package prompt

import (
    "context"
)

// Metric defines an evaluation function for prompt optimization
type Metric interface {
    // Score evaluates a prediction against gold truth
    // Returns score (0-1) and optional feedback for GEPA reflection
    Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error)
}

// ScoreWithFeedback combines numeric score with textual feedback for GEPA
type ScoreWithFeedback struct {
    Score    float64
    Feedback string
}

// Common metrics
type ExactMatchMetric struct{}

func (m *ExactMatchMetric) Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error) {
    expected := gold.Outputs["answer"]
    actual := pred.Outputs["answer"]

    if expected == actual {
        return ScoreWithFeedback{Score: 1.0, Feedback: "Correct!"}, nil
    }

    return ScoreWithFeedback{
        Score: 0.0,
        Feedback: fmt.Sprintf("Expected: %v, Got: %v", expected, actual),
    }, nil
}

// SemanticSimilarityMetric uses embeddings for soft matching
type SemanticSimilarityMetric struct {
    embedService ports.EmbeddingService
    threshold    float64
}

func (m *SemanticSimilarityMetric) Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error) {
    expected := gold.Outputs["answer"].(string)
    actual := pred.Outputs["answer"].(string)

    // Get embeddings
    embeddings, err := m.embedService.EmbedTexts(ctx, []string{expected, actual})
    if err != nil {
        return ScoreWithFeedback{}, err
    }

    // Calculate cosine similarity
    similarity := cosineSimilarity(embeddings[0], embeddings[1])

    feedback := fmt.Sprintf(
        "Semantic similarity: %.2f\nExpected: %s\nActual: %s",
        similarity, expected, actual,
    )

    return ScoreWithFeedback{Score: similarity, Feedback: feedback}, nil
}

// LLMJudgeMetric uses an LLM to evaluate response quality
type LLMJudgeMetric struct {
    llmService ports.LLMService
    criteria   string
}

func (m *LLMJudgeMetric) Score(ctx context.Context, gold, pred Example, trace *Trace) (ScoreWithFeedback, error) {
    prompt := fmt.Sprintf(`Evaluate this response based on: %s

Question: %v
Expected Answer: %v
Actual Response: %v

Provide a score from 0.0 to 1.0 and explain your reasoning.
Format: SCORE: X.X
REASONING: ...`,
        m.criteria,
        gold.Inputs["question"],
        gold.Outputs["answer"],
        pred.Outputs["answer"],
    )

    resp, err := m.llmService.Chat(ctx, []ports.LLMMessage{
        {Role: "user", Content: prompt},
    })
    if err != nil {
        return ScoreWithFeedback{}, err
    }

    score, reasoning := parseJudgeResponse(resp.Content)
    return ScoreWithFeedback{Score: score, Feedback: reasoning}, nil
}
```

### Phase 6: HTTP API & CLI

```go
// internal/adapters/http/handlers/optimization.go
package handlers

type OptimizationHandler struct {
    optService *services.OptimizationService
}

// POST /api/v1/optimizations
func (h *OptimizationHandler) CreateOptimization(w http.ResponseWriter, r *http.Request) {
    var req CreateOptimizationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, err)
        return
    }

    // Start optimization in background
    go func() {
        ctx := context.Background()
        h.optService.OptimizeSignature(
            ctx,
            req.Signature,
            req.TrainSet,
            req.ValSet,
            req.Metric,
            req.Config,
        )
    }()

    respondJSON(w, http.StatusAccepted, map[string]string{
        "status": "optimization_started",
    })
}

// GET /api/v1/optimizations/:id
func (h *OptimizationHandler) GetOptimization(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    run, err := h.optService.GetRun(r.Context(), id)
    if err != nil {
        respondError(w, http.StatusNotFound, err)
        return
    }
    respondJSON(w, http.StatusOK, run)
}
```

## Database Schema

```sql
-- Optimization runs
CREATE TABLE prompt_optimization_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signature_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    config JSONB NOT NULL,
    best_score FLOAT,
    iterations INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Prompt candidates (Pareto frontier)
CREATE TABLE prompt_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES prompt_optimization_runs(id),
    instructions TEXT NOT NULL,
    demos JSONB,
    coverage INT DEFAULT 0,
    avg_score FLOAT,
    generation INT NOT NULL DEFAULT 0,
    parent_id UUID REFERENCES prompt_candidates(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_candidates_run ON prompt_candidates(run_id);
CREATE INDEX idx_candidates_score ON prompt_candidates(avg_score DESC);

-- Evaluation results
CREATE TABLE prompt_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id UUID NOT NULL REFERENCES prompt_candidates(id),
    example_id VARCHAR(255) NOT NULL,
    score FLOAT NOT NULL,
    feedback TEXT,
    trace JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_evaluations_candidate ON prompt_evaluations(candidate_id);

-- Optimized programs (final results)
CREATE TABLE optimized_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES prompt_optimization_runs(id),
    signature_name VARCHAR(255) NOT NULL,
    instructions TEXT NOT NULL,
    demos JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_optimized_signature ON optimized_programs(signature_name);
```

## Configuration

### Multi-LLM Configuration

GEPA requires two LLM configurations:
- **Student Model**: The model being optimized (executes tasks)
- **Reflection Model**: Strong model for analyzing failures and proposing improvements

```yaml
# config.yaml additions
prompt_optimization:
  enabled: true

  # GEPA optimizer settings
  gepa:
    default_budget: "medium"  # light, medium, heavy
    num_threads: 8
    reflection_minibatch_size: 5
    skip_perfect_score: true
    use_merge: true

  # Reflection model (strong model for GEPA analysis)
  reflection_model:
    provider: "anthropic"
    model: "claude-3-5-sonnet-20241022"
    api_key: "${ANTHROPIC_API_KEY}"
    # Optional: custom endpoint for self-hosted models
    # base_url: "https://api.anthropic.com"

  # Student model (model being optimized)
  student_model:
    provider: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    # Optional: custom endpoint
    # base_url: "https://api.openai.com/v1"

  storage:
    type: "postgres"
    cache_optimized_programs: true
```

### Go Configuration Struct

```go
// internal/config/config.go additions

type PromptOptimizationConfig struct {
    Enabled         bool             `yaml:"enabled"`
    GEPA            GEPAConfig       `yaml:"gepa"`
    ReflectionModel LLMProviderConfig `yaml:"reflection_model"`
    StudentModel    LLMProviderConfig `yaml:"student_model"`
    Storage         StorageConfig    `yaml:"storage"`
}

type GEPAConfig struct {
    DefaultBudget           string `yaml:"default_budget"`
    NumThreads              int    `yaml:"num_threads"`
    ReflectionMinibatchSize int    `yaml:"reflection_minibatch_size"`
    SkipPerfectScore        bool   `yaml:"skip_perfect_score"`
    UseMerge                bool   `yaml:"use_merge"`
}

type LLMProviderConfig struct {
    Provider string `yaml:"provider"` // "openai", "anthropic", "google", "ollama"
    Model    string `yaml:"model"`
    APIKey   string `yaml:"api_key"`
    BaseURL  string `yaml:"base_url,omitempty"`
}

type StorageConfig struct {
    Type                   string `yaml:"type"`
    CacheOptimizedPrograms bool   `yaml:"cache_optimized_programs"`
}
```

### LLM Provider Factory

```go
// internal/prompt/llm_factory.go
package prompt

import (
    "github.com/XiaoConstantine/dspy-go/llms"
)

func NewLLMFromConfig(cfg LLMProviderConfig) (llms.LLM, error) {
    switch cfg.Provider {
    case "openai":
        return llms.NewOpenAI(
            llms.WithModel(cfg.Model),
            llms.WithAPIKey(cfg.APIKey),
        )
    case "anthropic":
        return llms.NewAnthropic(
            llms.WithModel(cfg.Model),
            llms.WithAPIKey(cfg.APIKey),
        )
    case "google":
        return llms.NewGoogle(
            llms.WithModel(cfg.Model),
            llms.WithAPIKey(cfg.APIKey),
        )
    case "ollama":
        return llms.NewOllama(
            llms.WithModel(cfg.Model),
            llms.WithBaseURL(cfg.BaseURL),
        )
    default:
        return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
    }
}
```

## Use Cases for Alicia

### 1. Response Quality Optimization

```go
// Optimize Alicia's conversational responses
sig := prompt.MustParseSignature("context, user_message, memories -> response")

trainset := loadConversationExamples("data/conversations_train.json")
valset := loadConversationExamples("data/conversations_val.json")

metric := &prompt.LLMJudgeMetric{
    llmService: reflectionLLM,
    criteria: "helpfulness, accuracy, conversational tone, use of memories",
}

optimized, _ := optService.OptimizeSignature(ctx, sig, trainset, valset, metric, config)
```

### 2. Tool Selection Optimization

```go
// Optimize when to use which tools
sig := prompt.MustParseSignature("user_intent, available_tools -> selected_tool, reasoning")

metric := &prompt.CompositeMetric{
    Components: []prompt.WeightedMetric{
        {Metric: &prompt.ExactMatchMetric{}, Weight: 0.7},
        {Metric: &prompt.ReasoningQualityMetric{}, Weight: 0.3},
    },
}
```

### 3. Memory Extraction Optimization

```go
// Optimize what to remember from conversations
sig := prompt.MustParseSignature("conversation -> key_facts: list[str], importance: int")

metric := &prompt.RecallMetric{
    // Measures if extracted facts cover expected key points
}
```

### 4. Tool Description Optimization

```go
// Optimize tool descriptions for better LLM understanding
pipeline := services.NewToolOptimizationPipeline(toolOptService, optService, toolService)

report, _ := pipeline.RunFullOptimization(ctx, GEPAConfig{
    Budget:     "medium",
    NumThreads: 8,
})

// Report contains:
// - Optimized descriptions per tool
// - Result formatting improvements
// - Tool selection accuracy before/after
fmt.Printf("Tool selection accuracy: %.1f%% -> %.1f%%\n",
    report.SelectionResults.OriginalAccuracy*100,
    report.SelectionResults.OptimizedAccuracy*100,
)
```

### 5. Tool Result Formatting Optimization

```go
// Optimize how tool results are presented to the LLM
sig := prompt.MustParseSignature(
    "tool_name, raw_result, user_context -> formatted_result, key_points: list[str]",
)

// Training data: (raw result, user question) -> (ideal formatted result, expected answer)
trainset := loadToolResultExamples("data/tool_results_train.json")

metric := &prompt.ToolResultMetric{
    llmService: reflectionLLM,
}

// Result: Learn optimal formatting per tool type
// - web_search: Extract titles, snippets, relevance
// - calculator: Show expression and result clearly
// - memory_query: Highlight relevant memories with context
```

### 6. Tool Argument Generation Optimization

```go
// Optimize how the LLM generates tool arguments
sig := prompt.MustParseSignature(
    "user_intent, tool_schema, conversation_context -> arguments: dict, validation_notes: str",
)

metric := &prompt.ToolArgumentMetric{}

// Learns patterns like:
// - "search for X" -> {"query": "X", "limit": 5}
// - "calculate X + Y" -> {"expression": "X + Y"}
// - "what did I say about X" -> {"query": "X", "limit": 3}
```

## Testing Strategy

### Unit Tests

```go
func TestGEPAOptimization(t *testing.T) {
    // Mock LLM service
    mockLLM := &MockLLMService{
        responses: map[string]string{
            "What is 2+2?": "4",
        },
    }

    sig := prompt.MustParseSignature("question -> answer")
    student := prompt.NewAliciaPredict(sig)

    trainset := []prompt.Example{
        {Inputs: map[string]any{"question": "What is 2+2?"}, Outputs: map[string]any{"answer": "4"}},
    }

    gepa := NewGEPA(metric, WithAuto("light"))
    optimized, err := gepa.Compile(ctx, student, trainset, trainset)

    assert.NoError(t, err)
    assert.NotNil(t, optimized)
}
```

### Integration Tests

```go
func TestOptimizationServiceE2E(t *testing.T) {
    // Use test database
    db := setupTestDB(t)
    defer db.Close()

    repo := postgres.NewOptimizationRepository(db)
    optService := services.NewOptimizationService(repo, llmService, reflectionLLM)

    sig := prompt.MustParseSignature("question -> answer")
    trainset := loadTestExamples(t, "testdata/qa_train.json")
    valset := loadTestExamples(t, "testdata/qa_val.json")

    result, err := optService.OptimizeSignature(ctx, sig, trainset, valset, metric, config)

    assert.NoError(t, err)
    assert.Greater(t, result.BestScore, 0.8)
}
```

## Implementation Phases Overview

| Phase | Description |
|-------|-------------|
| Phase 1 | Core integration with dspy-go |
| Phase 2 | GEPA optimization service |
| Phase 3 | Tool usage optimization (descriptions, results, selection) |
| Phase 4 | Memory-aware optimization |
| Phase 5 | Evaluation & metrics |
| Phase 6 | HTTP API & CLI |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| dspy-go API changes | Pin version, contribute upstream |
| Performance bottlenecks | Goroutines for parallel eval, caching |
| LLM rate limits | Backoff/retry, batching, caching |
| Complex metrics | Start simple, iterate based on needs |

## Integration with Frontend UX

This optimization system works in tandem with the [Frontend UX Enhancement Plan](frontend-ux-plan.md). User feedback collected through the frontend directly feeds into the optimization metrics:

### Feedback Sources

| Frontend Feature | Optimization Target |
|-----------------|---------------------|
| Message voting | Response quality metrics |
| Tool use voting | Tool selection/parameter optimization |
| Memory relevance voting | Memory retrieval scoring |
| Reasoning step voting | Chain-of-thought optimization |
| User notes | GEPA reflective feedback |

### Feedback Integration Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Frontend → Optimization Flow                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  User votes on response  ──→  Store in alicia_votes table       │
│           │                                                      │
│           ▼                                                      │
│  FeedbackMetric.Score()  ──→  Uses vote as ground truth         │
│           │                                                      │
│           ▼                                                      │
│  GEPA.Reflect()          ──→  Analyzes failures from feedback   │
│           │                                                      │
│           ▼                                                      │
│  Optimized prompts       ──→  Deployed to production            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### User Notes as Reflective Feedback

User notes from the frontend are particularly valuable for GEPA's reflective mutation:

```go
// Extract user notes for GEPA reflection
func (s *OptimizationService) gatherFeedback(ctx context.Context, messageID string) (*GEPAFeedback, error) {
    votes, _ := s.voteRepo.GetByMessage(ctx, messageID)
    notes, _ := s.noteRepo.GetByMessage(ctx, messageID)

    return &GEPAFeedback{
        Score:    calculateScore(votes),
        Feedback: combineNotes(notes), // Rich text for reflection
    }, nil
}
```

## Next Steps

1. **Foundation**: Add dspy-go dependency, create proof-of-concept
2. **Core Integration**: Implement Phase 1 (prompt abstraction layer)
3. **GEPA Service**: Implement Phase 2 (optimization service)
4. **Frontend Integration**: Connect UX feedback to optimization metrics
5. **Iteration**: Continuously improve based on evaluation results

## Related Documentation

- [Frontend UX Enhancement Plan](frontend-ux-plan.md) - User feedback collection
- [Architecture Overview](ARCHITECTURE.md) - System architecture
- [Database Schema](DATABASE.md) - Storage for optimization data
- [Protocol Specification](protocol/index.md) - Message format details

## References

- [DSPy Documentation](https://dspy.ai/)
- [GEPA Paper](https://arxiv.org/abs/2507.19457)
- [dspy-go Repository](https://github.com/XiaoConstantine/dspy-go)
- [LangChainGo](https://github.com/tmc/langchaingo)
