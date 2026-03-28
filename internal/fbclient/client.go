package fbclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const apiVersion = "2.12"

// Client is an HTTP client for the FlashBlade REST API.
type Client struct {
	BaseURL      string
	SessionToken string
	HTTPClient   *http.Client
}

// NewClient creates a new FlashBlade API client and authenticates.
func NewClient(fbURL, apiToken string, verifySSL bool) (*Client, error) {
	if !strings.HasPrefix(fbURL, "http") {
		fbURL = "https://" + fbURL
	}
	fbURL = strings.TrimRight(fbURL, "/")

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !verifySSL},
	}

	c := &Client{
		BaseURL:    fbURL,
		HTTPClient: &http.Client{Transport: transport},
	}

	if err := c.login(apiToken); err != nil {
		return nil, fmt.Errorf("FlashBlade login failed: %w", err)
	}

	return c, nil
}

func (c *Client) login(apiToken string) error {
	url := fmt.Sprintf("%s/api/login", c.BaseURL)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("api-token", apiToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login returned %d: %s", resp.StatusCode, string(body))
	}

	token := resp.Header.Get("x-auth-token")
	if token == "" {
		return fmt.Errorf("login succeeded but no x-auth-token header in response")
	}
	c.SessionToken = token
	return nil
}

// Close logs out from the FlashBlade API.
func (c *Client) Close() {
	if c.SessionToken == "" {
		return
	}
	url := fmt.Sprintf("%s/api/logout", c.BaseURL)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("x-auth-token", c.SessionToken)
	resp, err := c.HTTPClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	c.SessionToken = ""
}

// doRequest performs an authenticated HTTP request and returns the response body.
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, int, error) {
	url := fmt.Sprintf("%s/api/%s%s", c.BaseURL, apiVersion, path)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("x-auth-token", c.SessionToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return respBody, resp.StatusCode, nil
}

// apiResponse is the standard FlashBlade API response wrapper.
type apiResponse struct {
	Items []json.RawMessage `json:"items"`
}

// apiError represents a FlashBlade API error.
type apiError struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func parseAPIError(body []byte) string {
	var apiErr apiError
	if err := json.Unmarshal(body, &apiErr); err == nil && len(apiErr.Errors) > 0 {
		return apiErr.Errors[0].Message
	}
	return string(body)
}

// HumanToBytes converts a human-readable size string (e.g. "100G", "1T", "500M")
// to bytes. Supports B, K, M, G, T, P suffixes (case-insensitive).
// A plain number without suffix is treated as bytes.
func HumanToBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}

	re := regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*([bkmgtp]?)$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid size format: %q", s)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in size: %q", s)
	}

	multipliers := map[string]float64{
		"":  1,
		"b": 1,
		"k": 1024,
		"m": 1024 * 1024,
		"g": 1024 * 1024 * 1024,
		"t": 1024 * 1024 * 1024 * 1024,
		"p": 1024 * 1024 * 1024 * 1024 * 1024,
	}

	suffix := strings.ToLower(matches[2])
	return int64(value * multipliers[suffix]), nil
}

// BytesToHuman converts bytes to a human-readable string.
// Returns "" for 0 or negative values.
func BytesToHuman(n int64) string {
	if n <= 0 {
		return ""
	}
	units := []string{"", "K", "M", "G", "T", "P"}
	val := float64(n)
	for _, unit := range units {
		if val < 1024 {
			if val == float64(int64(val)) {
				return fmt.Sprintf("%d%s", int64(val), unit)
			}
			return fmt.Sprintf("%.1f%s", val, unit)
		}
		val /= 1024
	}
	return fmt.Sprintf("%.1fE", val)
}
