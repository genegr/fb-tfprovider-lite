package fbclient

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ObjectStoreAccount represents a FlashBlade S3 object store account.
type ObjectStoreAccount struct {
	ID               string `json:"id,omitempty"`
	Name             string `json:"name,omitempty"`
	QuotaLimit       *int64 `json:"quota_limit,omitempty"`
	HardLimitEnabled *bool  `json:"hard_limit_enabled,omitempty"`
	ObjectCount      int64  `json:"object_count,omitempty"`
	Created          int64  `json:"created,omitempty"`
}

// GetObjectStoreAccount reads an S3 account by name. Returns nil if not found.
func (c *Client) GetObjectStoreAccount(name string) (*ObjectStoreAccount, error) {
	path := fmt.Sprintf("/object-store-accounts?names=%s", url.QueryEscape(name))
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
		var acct ObjectStoreAccount
		if err := json.Unmarshal(resp.Items[0], &acct); err != nil {
			return nil, fmt.Errorf("failed to parse account: %w", err)
		}
		return &acct, nil
	}

	// 400 with "does not exist" means not found
	if status == 400 {
		return nil, nil
	}

	return nil, fmt.Errorf("get account %q failed (%d): %s", name, status, parseAPIError(body))
}

// CreateObjectStoreAccount creates a new S3 account.
func (c *Client) CreateObjectStoreAccount(name string) (*ObjectStoreAccount, error) {
	path := fmt.Sprintf("/object-store-accounts?names=%s", url.QueryEscape(name))
	body, status, err := c.doRequest("POST", path, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("create account %q failed (%d): %s", name, status, parseAPIError(body))
	}

	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("create account %q returned empty response", name)
	}
	var acct ObjectStoreAccount
	if err := json.Unmarshal(resp.Items[0], &acct); err != nil {
		return nil, fmt.Errorf("failed to parse account: %w", err)
	}
	return &acct, nil
}

// UpdateObjectStoreAccount updates an S3 account's settings.
func (c *Client) UpdateObjectStoreAccount(name string, patch map[string]interface{}) (*ObjectStoreAccount, error) {
	path := fmt.Sprintf("/object-store-accounts?names=%s", url.QueryEscape(name))
	body, status, err := c.doRequest("PATCH", path, patch)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("update account %q failed (%d): %s", name, status, parseAPIError(body))
	}

	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("update account %q returned empty response", name)
	}
	var acct ObjectStoreAccount
	if err := json.Unmarshal(resp.Items[0], &acct); err != nil {
		return nil, fmt.Errorf("failed to parse account: %w", err)
	}
	return &acct, nil
}

// DeleteObjectStoreAccount deletes an S3 account by name.
func (c *Client) DeleteObjectStoreAccount(name string) error {
	path := fmt.Sprintf("/object-store-accounts?names=%s", url.QueryEscape(name))
	body, status, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("delete account %q failed (%d): %s", name, status, parseAPIError(body))
	}
	return nil
}
