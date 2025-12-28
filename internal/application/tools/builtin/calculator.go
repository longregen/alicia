package builtin

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// RegisterCalculator registers the calculator tool with the tool service
func RegisterCalculator(ctx context.Context, toolService ports.ToolService) error {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "The mathematical expression to evaluate (e.g., '2 + 2', '10 * 5', 'sqrt(16)')",
			},
		},
		"required": []string{"expression"},
	}

	tool, err := toolService.EnsureTool(
		ctx,
		"calculator",
		"Evaluates mathematical expressions. Supports basic operations (+, -, *, /), exponentiation (^), and functions like sqrt, sin, cos, tan, log, ln, abs, ceil, floor.",
		schema,
	)
	if err != nil {
		return fmt.Errorf("failed to register calculator tool: %w", err)
	}

	// Register the executor
	err = toolService.RegisterExecutor("calculator", func(ctx context.Context, arguments map[string]any) (any, error) {
		expression, ok := arguments["expression"].(string)
		if !ok {
			return nil, fmt.Errorf("expression must be a string")
		}

		result, err := evaluateExpression(expression)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expression: %w", err)
		}

		return map[string]any{
			"expression": expression,
			"result":     result,
		}, nil
	})

	if err != nil {
		return fmt.Errorf("failed to register calculator executor: %w", err)
	}

	log.Printf("Registered calculator tool: %s", tool.ID)
	return nil
}

// evaluateExpression evaluates a simple mathematical expression
// This is a basic implementation. For production, consider using a proper expression parser.
func evaluateExpression(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)
	expr = strings.ToLower(expr)

	// Handle functions
	if strings.HasPrefix(expr, "sqrt(") && strings.HasSuffix(expr, ")") {
		inner := expr[5 : len(expr)-1]
		val, err := evaluateExpression(inner)
		if err != nil {
			return 0, err
		}
		return math.Sqrt(val), nil
	}

	if strings.HasPrefix(expr, "abs(") && strings.HasSuffix(expr, ")") {
		inner := expr[4 : len(expr)-1]
		val, err := evaluateExpression(inner)
		if err != nil {
			return 0, err
		}
		return math.Abs(val), nil
	}

	if strings.HasPrefix(expr, "sin(") && strings.HasSuffix(expr, ")") {
		inner := expr[4 : len(expr)-1]
		val, err := evaluateExpression(inner)
		if err != nil {
			return 0, err
		}
		return math.Sin(val), nil
	}

	if strings.HasPrefix(expr, "cos(") && strings.HasSuffix(expr, ")") {
		inner := expr[4 : len(expr)-1]
		val, err := evaluateExpression(inner)
		if err != nil {
			return 0, err
		}
		return math.Cos(val), nil
	}

	if strings.HasPrefix(expr, "tan(") && strings.HasSuffix(expr, ")") {
		inner := expr[4 : len(expr)-1]
		val, err := evaluateExpression(inner)
		if err != nil {
			return 0, err
		}
		return math.Tan(val), nil
	}

	if strings.HasPrefix(expr, "log(") && strings.HasSuffix(expr, ")") {
		inner := expr[4 : len(expr)-1]
		val, err := evaluateExpression(inner)
		if err != nil {
			return 0, err
		}
		return math.Log10(val), nil
	}

	if strings.HasPrefix(expr, "ln(") && strings.HasSuffix(expr, ")") {
		inner := expr[3 : len(expr)-1]
		val, err := evaluateExpression(inner)
		if err != nil {
			return 0, err
		}
		return math.Log(val), nil
	}

	if strings.HasPrefix(expr, "ceil(") && strings.HasSuffix(expr, ")") {
		inner := expr[5 : len(expr)-1]
		val, err := evaluateExpression(inner)
		if err != nil {
			return 0, err
		}
		return math.Ceil(val), nil
	}

	if strings.HasPrefix(expr, "floor(") && strings.HasSuffix(expr, ")") {
		inner := expr[6 : len(expr)-1]
		val, err := evaluateExpression(inner)
		if err != nil {
			return 0, err
		}
		return math.Floor(val), nil
	}

	// Handle exponentiation
	if strings.Contains(expr, "^") {
		parts := strings.SplitN(expr, "^", 2)
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid exponentiation expression")
		}
		base, err := evaluateExpression(parts[0])
		if err != nil {
			return 0, err
		}
		exp, err := evaluateExpression(parts[1])
		if err != nil {
			return 0, err
		}
		return math.Pow(base, exp), nil
	}

	// Handle multiplication and division (higher precedence)
	for i, op := range []string{"*", "/"} {
		if strings.Contains(expr, op) {
			parts := strings.SplitN(expr, op, 2)
			if len(parts) != 2 {
				return 0, fmt.Errorf("invalid %s expression", op)
			}
			left, err := evaluateExpression(parts[0])
			if err != nil {
				return 0, err
			}
			right, err := evaluateExpression(parts[1])
			if err != nil {
				return 0, err
			}
			if i == 0 {
				return left * right, nil
			}
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		}
	}

	// Handle addition and subtraction (lower precedence)
	// Find the last occurrence to handle left-to-right evaluation
	for i, op := range []string{"+", "-"} {
		idx := strings.LastIndex(expr, op)
		if idx > 0 { // idx > 0 to avoid negative numbers at the start
			parts := []string{expr[:idx], expr[idx+1:]}
			left, err := evaluateExpression(parts[0])
			if err != nil {
				return 0, err
			}
			right, err := evaluateExpression(parts[1])
			if err != nil {
				return 0, err
			}
			if i == 0 {
				return left + right, nil
			}
			return left - right, nil
		}
	}

	// Try to parse as a number
	val, err := strconv.ParseFloat(expr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid expression: %s", expr)
	}

	return val, nil
}

// GetCalculatorTool returns the calculator tool definition
func GetCalculatorTool() *models.Tool {
	return &models.Tool{
		Name:        "calculator",
		Description: "Evaluates mathematical expressions",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expression": map[string]any{
					"type":        "string",
					"description": "The mathematical expression to evaluate",
				},
			},
			"required": []string{"expression"},
		},
		Enabled: true,
	}
}
