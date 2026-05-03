package cmd

import "github.com/spf13/cobra"

func newCacheCmd() *cobra.Command { return &cobra.Command{Use: "cache", Hidden: true, RunE: stubRun} }

func stubRun(_ *cobra.Command, _ []string) error { return nil }
