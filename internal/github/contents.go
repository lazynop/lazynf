package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
)

const (
	nerdFontsOwner = "ryanoasis"
	nerdFontsRepo  = "nerd-fonts"
)

// PatchedFontsList fetches the names of subdirectories of `patched-fonts/` in
// the upstream Nerd Fonts repository on the master branch. These directory
// names are the canonical font identifiers used in release asset names too.
//
// Returned slice is sorted alphabetically.
func (c *Client) PatchedFontsList() ([]string, error) {
	resp, err := c.do("GET", fmt.Sprintf(
		"/repos/%s/%s/contents/patched-fonts?ref=master",
		nerdFontsOwner, nerdFontsRepo,
	))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github contents returned %d: %s", resp.StatusCode, string(body))
	}

	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode contents: %w", err)
	}

	var fonts []string
	for _, e := range entries {
		if e.Type == "dir" {
			fonts = append(fonts, e.Name)
		}
	}
	sort.Strings(fonts)
	return fonts, nil
}
