package cmd

import "github.com/spf13/cobra"

func newInstallCmd() *cobra.Command {
	return &cobra.Command{Use: "install", Hidden: true, RunE: stubRun}
}
func newListCmd() *cobra.Command   { return &cobra.Command{Use: "list", Hidden: true, RunE: stubRun} }
func newSearchCmd() *cobra.Command { return &cobra.Command{Use: "search", Hidden: true, RunE: stubRun} }
func newCacheCmd() *cobra.Command  { return &cobra.Command{Use: "cache", Hidden: true, RunE: stubRun} }

func stubRun(_ *cobra.Command, _ []string) error { return nil }
