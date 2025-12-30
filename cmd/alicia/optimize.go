package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/llm"
	"github.com/longregen/alicia/internal/ports"
	"github.com/spf13/cobra"
)

// optimizeCmd provides subcommands for optimization management
func optimizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Manage prompt optimization runs",
		Long: `Manage DSPy/GEPA optimization runs for prompt engineering.

Subcommands:
  list     List all optimization runs
  show     Show details of a specific run
  run      Start a new optimization run
  candidates  List candidates for a run
  best     Show the best candidate for a run`,
	}

	cmd.AddCommand(
		optimizeListCmd(),
		optimizeShowCmd(),
		optimizeRunCmd(),
		optimizeCandidatesCmd(),
		optimizeBestCmd(),
	)

	return cmd
}

// optimizeListCmd lists all optimization runs
func optimizeListCmd() *cobra.Command {
	var status string
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List optimization runs",
		Long:  `List all optimization runs with optional filtering by status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			pool, err := initDB(ctx)
			if err != nil {
				return err
			}
			defer pool.Close()

			repo := postgres.NewOptimizationRepository(pool)
			idGen := id.New()
			optimizationConfig := services.DefaultOptimizationConfig()
			llmService := llm.NewService(llmClient)
			optimizationService := services.NewOptimizationService(
				repo,
				llmService,
				idGen,
				optimizationConfig,
			)

			opts := ports.ListOptimizationRunsOptions{}
			if status != "" {
				opts.Status = status
			}
			if limit > 0 {
				opts.Limit = limit
			}

			runs, err := optimizationService.ListOptimizationRuns(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list runs: %w", err)
			}

			if len(runs) == 0 {
				fmt.Println("No optimization runs found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSTATUS\tITERATIONS\tBEST SCORE\tSTARTED\tCOMPLETED")
			fmt.Fprintln(w, "--\t----\t------\t----------\t----------\t-------\t---------")

			for _, run := range runs {
				completedStr := "N/A"
				if run.CompletedAt != nil {
					completedStr = run.CompletedAt.Format("2006-01-02 15:04")
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%d/%d\t%.4f\t%s\t%s\n",
					run.ID[:8],
					run.Name,
					run.Status,
					run.Iterations,
					run.MaxIterations,
					run.BestScore,
					run.StartedAt.Format("2006-01-02 15:04"),
					completedStr,
				)
			}

			w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status (running, completed, failed)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of runs to list")

	return cmd
}

// optimizeShowCmd shows details of a specific optimization run
func optimizeShowCmd() *cobra.Command {
	var showJSON bool

	cmd := &cobra.Command{
		Use:   "show <run-id>",
		Short: "Show optimization run details",
		Long:  `Show detailed information about a specific optimization run.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			runID := args[0]

			pool, err := initDB(ctx)
			if err != nil {
				return err
			}
			defer pool.Close()

			repo := postgres.NewOptimizationRepository(pool)
			idGen := id.New()
			optimizationConfig := services.DefaultOptimizationConfig()
			llmService := llm.NewService(llmClient)
			optimizationService := services.NewOptimizationService(
				repo,
				llmService,
				idGen,
				optimizationConfig,
			)

			run, err := optimizationService.GetOptimizationRun(ctx, runID)
			if err != nil {
				return fmt.Errorf("failed to get run: %w", err)
			}

			if showJSON {
				data, err := json.MarshalIndent(run, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Optimization Run: %s\n", run.ID)
			fmt.Printf("Name:        %s\n", run.Name)
			fmt.Printf("Status:      %s\n", run.Status)
			fmt.Printf("Prompt Type: %s\n", run.PromptType)
			fmt.Printf("Iterations:  %d / %d\n", run.Iterations, run.MaxIterations)
			fmt.Printf("Best Score:  %.4f\n", run.BestScore)
			fmt.Printf("Started:     %s\n", run.StartedAt.Format(time.RFC3339))
			if run.CompletedAt != nil {
				fmt.Printf("Completed:   %s\n", run.CompletedAt.Format(time.RFC3339))
			}
			fmt.Println()

			if len(run.DimensionWeights) > 0 {
				fmt.Println("Dimension Weights:")
				for dim, weight := range run.DimensionWeights {
					fmt.Printf("  %s: %.2f\n", dim, weight)
				}
				fmt.Println()
			}

			if len(run.BestDimScores) > 0 {
				fmt.Println("Best Dimension Scores:")
				for dim, score := range run.BestDimScores {
					fmt.Printf("  %s: %.4f\n", dim, score)
				}
				fmt.Println()
			}

			if len(run.Config) > 0 {
				fmt.Println("Configuration:")
				for key, val := range run.Config {
					fmt.Printf("  %s: %v\n", key, val)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")

	return cmd
}

// optimizeRunCmd starts a new optimization run
func optimizeRunCmd() *cobra.Command {
	var (
		name           string
		promptType     string
		baselinePrompt string
		maxIterations  int
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start a new optimization run",
		Long: `Start a new DSPy/GEPA optimization run.

This command creates a new optimization run configuration. The actual optimization
will run asynchronously in the background.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if name == "" {
				return fmt.Errorf("name is required (use --name)")
			}
			if promptType == "" {
				return fmt.Errorf("prompt type is required (use --type)")
			}

			pool, err := initDB(ctx)
			if err != nil {
				return err
			}
			defer pool.Close()

			repo := postgres.NewOptimizationRepository(pool)
			idGen := id.New()
			optimizationConfig := services.DefaultOptimizationConfig()
			if maxIterations > 0 {
				optimizationConfig.MaxIterations = maxIterations
			}

			llmService := llm.NewService(llmClient)
			optimizationService := services.NewOptimizationService(
				repo,
				llmService,
				idGen,
				optimizationConfig,
			)

			run, err := optimizationService.StartOptimizationRun(
				ctx,
				name,
				promptType,
				baselinePrompt,
			)
			if err != nil {
				return fmt.Errorf("failed to start optimization run: %w", err)
			}

			fmt.Printf("Optimization run created: %s\n", run.ID)
			fmt.Printf("Name: %s\n", run.Name)
			fmt.Printf("Type: %s\n", run.PromptType)
			fmt.Printf("Max Iterations: %d\n", run.MaxIterations)
			fmt.Printf("Status: %s\n", run.Status)
			fmt.Println()
			fmt.Println("Note: This creates a run configuration. To execute the optimization,")
			fmt.Println("you need to call the optimization service programmatically or via the API.")

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Name of the optimization run (required)")
	cmd.Flags().StringVarP(&promptType, "type", "t", "", "Prompt type (required)")
	cmd.Flags().StringVarP(&baselinePrompt, "baseline", "b", "", "Baseline prompt text")
	cmd.Flags().IntVarP(&maxIterations, "iterations", "i", 100, "Maximum iterations")

	return cmd
}

// optimizeCandidatesCmd lists candidates for a run
func optimizeCandidatesCmd() *cobra.Command {
	var showJSON bool

	cmd := &cobra.Command{
		Use:   "candidates <run-id>",
		Short: "List prompt candidates for a run",
		Long:  `List all prompt candidates generated during an optimization run.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			runID := args[0]

			pool, err := initDB(ctx)
			if err != nil {
				return err
			}
			defer pool.Close()

			repo := postgres.NewOptimizationRepository(pool)
			idGen := id.New()
			optimizationConfig := services.DefaultOptimizationConfig()
			optimizationService := services.NewOptimizationService(
				repo,
				llm.NewService(llmClient),
				idGen,
				optimizationConfig,
			)

			candidates, err := optimizationService.GetCandidates(ctx, runID)
			if err != nil {
				return fmt.Errorf("failed to get candidates: %w", err)
			}

			if len(candidates) == 0 {
				fmt.Println("No candidates found for this run.")
				return nil
			}

			if showJSON {
				data, err := json.MarshalIndent(candidates, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tITERATION\tSCORE\tEVALS\tSUCCESS\tCREATED")
			fmt.Fprintln(w, "--\t---------\t-----\t-----\t-------\t-------")

			for _, candidate := range candidates {
				successRate := 0.0
				if candidate.EvaluationCount > 0 {
					successRate = float64(candidate.SuccessCount) / float64(candidate.EvaluationCount) * 100
				}

				fmt.Fprintf(w, "%s\t%d\t%.4f\t%d\t%.1f%%\t%s\n",
					candidate.ID[:8],
					candidate.Iteration,
					candidate.Score,
					candidate.EvaluationCount,
					successRate,
					candidate.CreatedAt.Format("2006-01-02 15:04"),
				)
			}

			w.Flush()
			return nil
		},
	}

	cmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")

	return cmd
}

// optimizeBestCmd shows the best candidate for a run
func optimizeBestCmd() *cobra.Command {
	var showPrompt bool

	cmd := &cobra.Command{
		Use:   "best <run-id>",
		Short: "Show the best candidate for a run",
		Long:  `Show the best performing prompt candidate for an optimization run.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			runID := args[0]

			pool, err := initDB(ctx)
			if err != nil {
				return err
			}
			defer pool.Close()

			repo := postgres.NewOptimizationRepository(pool)
			idGen := id.New()
			optimizationConfig := services.DefaultOptimizationConfig()
			optimizationService := services.NewOptimizationService(
				repo,
				llm.NewService(llmClient),
				idGen,
				optimizationConfig,
			)

			candidate, err := optimizationService.GetBestCandidate(ctx, runID)
			if err != nil {
				return fmt.Errorf("failed to get best candidate: %w", err)
			}

			fmt.Printf("Best Candidate: %s\n", candidate.ID)
			fmt.Printf("Iteration:      %d\n", candidate.Iteration)
			fmt.Printf("Score:          %.4f\n", candidate.Score)
			fmt.Printf("Evaluations:    %d\n", candidate.EvaluationCount)
			fmt.Printf("Success Rate:   %d/%d (%.1f%%)\n",
				candidate.SuccessCount,
				candidate.EvaluationCount,
				float64(candidate.SuccessCount)/float64(candidate.EvaluationCount)*100,
			)
			fmt.Println()

			if len(candidate.DimensionScores) > 0 {
				fmt.Println("Dimension Scores:")
				dims := []string{"successRate", "quality", "efficiency", "robustness", "generalization", "diversity", "innovation"}
				for _, dim := range dims {
					if score, ok := candidate.DimensionScores[dim]; ok {
						fmt.Printf("  %s: %.4f\n", dim, score)
					}
				}
				fmt.Println()
			}

			if showPrompt {
				fmt.Println("Prompt Text:")
				fmt.Println("---")
				fmt.Println(candidate.PromptText)
				fmt.Println("---")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showPrompt, "prompt", "p", false, "Show the full prompt text")

	return cmd
}
