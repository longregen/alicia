package langfuse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"
)

// ScoreStats holds aggregate statistics for a score.
type ScoreStats struct {
	Name    string
	Count   int
	Average float64
	Min     float64
	Max     float64
	StdDev  float64
}

// scoreListResponse represents the API response for listing scores.
type scoreListResponse struct {
	Data []scoreItem `json:"data"`
	Meta struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		TotalItems int `json:"totalItems"`
		TotalPages int `json:"totalPages"`
	} `json:"meta"`
}

type scoreItem struct {
	ID        string    `json:"id"`
	TraceID   string    `json:"traceId"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	DataType  string    `json:"dataType"`
}

// GetScoreStats fetches aggregate statistics for a score from Langfuse.
// It queries scores since the given time and calculates statistics locally.
func (c *Client) GetScoreStats(ctx context.Context, scoreName string, since time.Time) (*ScoreStats, error) {
	// Fetch scores with pagination
	// Note: Langfuse API may have limits on how far back we can query
	scores, err := c.fetchScores(ctx, scoreName, since)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch scores: %w", err)
	}

	if len(scores) == 0 {
		return &ScoreStats{
			Name:    scoreName,
			Count:   0,
			Average: 0,
			Min:     0,
			Max:     0,
			StdDev:  0,
		}, nil
	}

	// Calculate statistics
	stats := &ScoreStats{
		Name:  scoreName,
		Count: len(scores),
		Min:   scores[0].Value,
		Max:   scores[0].Value,
	}

	sum := 0.0
	for _, s := range scores {
		sum += s.Value
		if s.Value < stats.Min {
			stats.Min = s.Value
		}
		if s.Value > stats.Max {
			stats.Max = s.Value
		}
	}
	stats.Average = sum / float64(len(scores))

	// Calculate standard deviation
	sumSquaredDiff := 0.0
	for _, s := range scores {
		diff := s.Value - stats.Average
		sumSquaredDiff += diff * diff
	}
	stats.StdDev = math.Sqrt(sumSquaredDiff / float64(len(scores)))

	return stats, nil
}

// fetchScores retrieves scores from Langfuse API with the given filter.
func (c *Client) fetchScores(ctx context.Context, scoreName string, since time.Time) ([]scoreItem, error) {
	var allScores []scoreItem
	page := 1
	limit := 100

	for {
		apiURL := fmt.Sprintf("https://%s/api/public/scores", c.host)
		params := url.Values{}
		params.Set("name", scoreName)
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("limit", fmt.Sprintf("%d", limit))
		if !since.IsZero() {
			params.Set("fromTimestamp", since.Format(time.RFC3339))
		}
		apiURL += "?" + params.Encode()

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.SetBasicAuth(c.publicKey, c.secretKey)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		var apiResp scoreListResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		allScores = append(allScores, apiResp.Data...)

		// Check if we've fetched all pages
		if page >= apiResp.Meta.TotalPages || len(apiResp.Data) == 0 {
			break
		}
		page++
	}

	return allScores, nil
}

// CheckScoreRegression compares recent scores against a baseline period.
// Returns true if there's a significant regression (average dropped by more than threshold).
// The threshold is expressed as an absolute value (e.g., 0.5 means a drop of 0.5 or more).
func CheckScoreRegression(ctx context.Context, client *Client, scoreName string, threshold float64) (bool, error) {
	if client == nil {
		return false, fmt.Errorf("client is nil")
	}

	// Compare last 24 hours against the previous 7 days
	now := time.Now()
	recentStart := now.Add(-24 * time.Hour)
	baselineStart := now.Add(-7 * 24 * time.Hour)

	// Get recent stats (last 24 hours)
	recentStats, err := client.GetScoreStats(ctx, scoreName, recentStart)
	if err != nil {
		return false, fmt.Errorf("failed to get recent stats: %w", err)
	}

	// Get baseline stats (previous 7 days, excluding recent)
	baselineScores, err := client.fetchScores(ctx, scoreName, baselineStart)
	if err != nil {
		return false, fmt.Errorf("failed to fetch baseline scores: %w", err)
	}

	// Filter baseline to exclude recent period
	var filteredBaseline []scoreItem
	for _, s := range baselineScores {
		if s.Timestamp.Before(recentStart) {
			filteredBaseline = append(filteredBaseline, s)
		}
	}

	// Need enough data points for comparison
	if recentStats.Count < 5 {
		client.log.Printf("langfuse: not enough recent data for regression check on %q (%d samples)", scoreName, recentStats.Count)
		return false, nil
	}
	if len(filteredBaseline) < 10 {
		client.log.Printf("langfuse: not enough baseline data for regression check on %q (%d samples)", scoreName, len(filteredBaseline))
		return false, nil
	}

	// Calculate baseline average
	baselineSum := 0.0
	for _, s := range filteredBaseline {
		baselineSum += s.Value
	}
	baselineAvg := baselineSum / float64(len(filteredBaseline))

	// Check for regression
	drop := baselineAvg - recentStats.Average
	isRegression := drop > threshold

	if isRegression {
		client.log.Printf("langfuse: REGRESSION detected for %q: baseline=%.3f, recent=%.3f, drop=%.3f (threshold=%.3f)",
			scoreName, baselineAvg, recentStats.Average, drop, threshold)
	}

	return isRegression, nil
}

// ScoreRegressionReport holds the result of checking multiple scores for regression.
type ScoreRegressionReport struct {
	CheckedAt   time.Time
	Regressions []ScoreRegression
	Errors      []string
}

// ScoreRegression represents a single detected regression.
type ScoreRegression struct {
	ScoreName    string
	BaselineAvg  float64
	RecentAvg    float64
	Drop         float64
	Threshold    float64
}

// CheckAllScoreRegressions checks all standard Alicia scores for regression.
// Returns a report with any detected regressions.
func CheckAllScoreRegressions(ctx context.Context, client *Client, threshold float64) (*ScoreRegressionReport, error) {
	if client == nil {
		return nil, fmt.Errorf("client is nil")
	}

	report := &ScoreRegressionReport{
		CheckedAt: time.Now(),
	}

	scoreNames := GetAllScoreConfigNames()
	for _, name := range scoreNames {
		hasRegression, err := CheckScoreRegression(ctx, client, name, threshold)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}

		if hasRegression {
			// Get the actual values for the report
			now := time.Now()
			recentStart := now.Add(-24 * time.Hour)
			baselineStart := now.Add(-7 * 24 * time.Hour)

			recentStats, _ := client.GetScoreStats(ctx, name, recentStart)
			baselineScores, _ := client.fetchScores(ctx, name, baselineStart)

			var baselineAvg float64
			count := 0
			for _, s := range baselineScores {
				if s.Timestamp.Before(recentStart) {
					baselineAvg += s.Value
					count++
				}
			}
			if count > 0 {
				baselineAvg /= float64(count)
			}

			report.Regressions = append(report.Regressions, ScoreRegression{
				ScoreName:   name,
				BaselineAvg: baselineAvg,
				RecentAvg:   recentStats.Average,
				Drop:        baselineAvg - recentStats.Average,
				Threshold:   threshold,
			})
		}
	}

	return report, nil
}
