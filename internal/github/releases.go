package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// LatestTag fetches the tag_name of the latest published release of owner/repo.
func (c *Client) LatestTag(owner, repo string) (string, error) {
	resp, err := c.do("GET", fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github releases/latest returned %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode release: %w", err)
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("github returned empty tag_name")
	}
	return payload.TagName, nil
}
