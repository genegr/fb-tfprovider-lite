package fbclient

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// Bucket represents a FlashBlade S3 bucket.
type Bucket struct {
	ID               string          `json:"id,omitempty"`
	Name             string          `json:"name,omitempty"`
	Account          *BucketAccount  `json:"account,omitempty"`
	BucketType       string          `json:"bucket_type,omitempty"`
	Created          int64           `json:"created,omitempty"`
	Destroyed        bool            `json:"destroyed,omitempty"`
	HardLimitEnabled *bool           `json:"hard_limit_enabled,omitempty"`
	ObjectCount      int64           `json:"object_count,omitempty"`
	QuotaLimit       *int64          `json:"quota_limit,omitempty"`
	Versioning       string          `json:"versioning,omitempty"`
}

// BucketAccount is the account reference within a Bucket response.
type BucketAccount struct {
	Name string `json:"name,omitempty"`
}

// BucketCreateBody is the request body for creating a bucket.
type BucketCreateBody struct {
	Account          *BucketAccount `json:"account,omitempty"`
	QuotaLimit       *string        `json:"quota_limit,omitempty"`
	HardLimitEnabled *bool          `json:"hard_limit_enabled,omitempty"`
}

// GetBucket reads a bucket by name. Returns nil if not found or destroyed.
func (c *Client) GetBucket(name string) (*Bucket, error) {
	path := fmt.Sprintf("/buckets?names=%s", url.QueryEscape(name))
	body, status, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if status == 200 {
		var resp apiResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		if len(resp.Items) == 0 {
			return nil, nil
		}
		var bucket Bucket
		if err := json.Unmarshal(resp.Items[0], &bucket); err != nil {
			return nil, fmt.Errorf("failed to parse bucket: %w", err)
		}
		return &bucket, nil
	}

	if status == 400 {
		return nil, nil
	}

	return nil, fmt.Errorf("get bucket %q failed (%d): %s", name, status, parseAPIError(body))
}

// CreateBucket creates a new S3 bucket.
func (c *Client) CreateBucket(name string, createBody BucketCreateBody) (*Bucket, error) {
	path := fmt.Sprintf("/buckets?names=%s", url.QueryEscape(name))
	body, status, err := c.doRequest("POST", path, createBody)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("create bucket %q failed (%d): %s", name, status, parseAPIError(body))
	}

	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("create bucket %q returned empty response", name)
	}
	var bucket Bucket
	if err := json.Unmarshal(resp.Items[0], &bucket); err != nil {
		return nil, fmt.Errorf("failed to parse bucket: %w", err)
	}
	return &bucket, nil
}

// UpdateBucket updates a bucket's settings.
func (c *Client) UpdateBucket(name string, patch map[string]interface{}) (*Bucket, error) {
	path := fmt.Sprintf("/buckets?names=%s", url.QueryEscape(name))
	body, status, err := c.doRequest("PATCH", path, patch)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("update bucket %q failed (%d): %s", name, status, parseAPIError(body))
	}

	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("update bucket %q returned empty response", name)
	}
	var bucket Bucket
	if err := json.Unmarshal(resp.Items[0], &bucket); err != nil {
		return nil, fmt.Errorf("failed to parse bucket: %w", err)
	}
	return &bucket, nil
}

// DeleteBucket soft-deletes a bucket (marks as destroyed).
// If eradicate is true, also permanently removes the bucket.
func (c *Client) DeleteBucket(name string, eradicate bool) error {
	// Step 1: soft-delete
	patch := map[string]interface{}{"destroyed": true}
	path := fmt.Sprintf("/buckets?names=%s", url.QueryEscape(name))
	body, status, err := c.doRequest("PATCH", path, patch)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("soft-delete bucket %q failed (%d): %s", name, status, parseAPIError(body))
	}

	// Step 2: eradicate if requested
	if eradicate {
		body, status, err = c.doRequest("DELETE", path, nil)
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("eradicate bucket %q failed (%d): %s", name, status, parseAPIError(body))
		}
	}

	return nil
}
