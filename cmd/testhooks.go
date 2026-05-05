package cmd

import (
	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/fonts"
	"github.com/lazynop/lazynf/internal/github"
)

// These overrides exist for E2E tests under cmd/ that need to swap network/IO
// dependencies. They are package-level vars deliberately: keeping the override
// surface tiny avoids bleeding test concerns into production code paths.

var (
	testGitHubBaseURL string
	testAssetURLBase  string
	testRefresher     fontcache.Refresher
)

// SetTestGitHubBaseURL points the github.Client at a custom base URL.
func SetTestGitHubBaseURL(u string) { testGitHubBaseURL = u }

// SetTestAssetURLBase overrides the release-asset download base URL.
func SetTestAssetURLBase(u string) { testAssetURLBase = u }

// SetTestRefresher swaps the fontcache.Refresher (e.g. with a FakeRefresher).
func SetTestRefresher(r fontcache.Refresher) { testRefresher = r }

// ResetTestOverrides clears all test hooks (call from defer).
func ResetTestOverrides() {
	testGitHubBaseURL = ""
	testAssetURLBase = ""
	testRefresher = nil
}

func newGitHubClient() *github.Client {
	c := github.NewClient()
	if testGitHubBaseURL != "" {
		c.BaseURL = testGitHubBaseURL
	}
	return c
}

func assetURLBase() string {
	if testAssetURLBase != "" {
		return testAssetURLBase
	}
	return fonts.DefaultAssetURLBase
}

func refresher() fontcache.Refresher {
	if testRefresher != nil {
		return testRefresher
	}
	return fontcache.Default()
}
