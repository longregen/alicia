package speech

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/longregen/alicia/internal/adapters/retry"
)

type Client struct {
	httpClient  *http.Client
	baseURL     string
	retryConfig retry.BackoffConfig
}

func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Default request timeout
		},
		baseURL:     baseURL,
		retryConfig: retry.HTTPConfig(),
	}
}

func (c *Client) PostJSON(ctx context.Context, endpoint string, payload interface{}, response interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	var respBody []byte
	var statusCode int

	err = retry.WithBackoffHTTP(ctx, c.retryConfig, func() (int, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		statusCode = resp.StatusCode
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return statusCode, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return statusCode, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		return statusCode, nil
	})

	if err != nil {
		return err
	}

	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) PostJSONRaw(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var respBody []byte
	var statusCode int

	err = retry.WithBackoffHTTP(ctx, c.retryConfig, func() (int, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		statusCode = resp.StatusCode
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return statusCode, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return statusCode, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		return statusCode, nil
	})

	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func (c *Client) PostMultipart(ctx context.Context, endpoint string, fields map[string]string, fileField string, fileName string, fileData []byte, response interface{}) error {
	var respBody []byte
	var statusCode int

	err := retry.WithBackoffHTTP(ctx, c.retryConfig, func() (int, error) {
		// Rebuild multipart body for each retry attempt
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Add form fields
		for key, val := range fields {
			if err := writer.WriteField(key, val); err != nil {
				return 0, fmt.Errorf("failed to write field %s: %w", key, err)
			}
		}

		// Add file
		if fileField != "" && fileData != nil {
			part, err := writer.CreateFormFile(fileField, fileName)
			if err != nil {
				return 0, fmt.Errorf("failed to create form file: %w", err)
			}
			if _, err := part.Write(fileData); err != nil {
				return 0, fmt.Errorf("failed to write file data: %w", err)
			}
		}

		if err := writer.Close(); err != nil {
			return 0, fmt.Errorf("failed to close multipart writer: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, &buf)
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		statusCode = resp.StatusCode
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return statusCode, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return statusCode, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		return statusCode, nil
	})

	if err != nil {
		return err
	}

	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) Get(ctx context.Context, endpoint string, response interface{}) error {
	var respBody []byte
	var statusCode int

	err := retry.WithBackoffHTTP(ctx, c.retryConfig, func() (int, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		statusCode = resp.StatusCode
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return statusCode, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return statusCode, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		return statusCode, nil
	})

	if err != nil {
		return err
	}

	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
