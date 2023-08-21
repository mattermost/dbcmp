package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/mattermost/dbcmp/internal/store"
	"github.com/spf13/cobra"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:     "dbcmp",
		Short:   "dbcmp is your go-to database content comparison tool",
		Long:    "dbcmp is a powerful and efficient command line tool designed to simplify the process of comparing content between two databases.",
		RunE:    runRootCmdFn,
		Version: versionCmdFn(),
	}

	rootCmd.PersistentFlags().String("source", "", "source database dsn")
	rootCmd.PersistentFlags().String("target", "", "target database dsn")
	rootCmd.Flags().StringSlice("exclude", []string{}, "exclude tables from comparison, takes comma-separated values.")
	rootCmd.Flags().Int("page-size", 1000, "page size for each checksum comparison.")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runRootCmdFn(cmd *cobra.Command, args []string) error {
	source, err := cmd.Flags().GetString("source")
	if err != nil {
		return err
	}

	target, err := cmd.Flags().GetString("target")
	if err != nil {
		return err
	}

	excl, err := cmd.Flags().GetStringSlice("exclude")
	if err != nil {
		return err
	}

	pageSize, err := cmd.Flags().GetInt("page-size")
	if err != nil {
		return err
	}

	if pageSize < 2 {
		return fmt.Errorf("page size could not be less than 2 (two), current value is: %d", pageSize)
	}

	diffs, err := store.Compare(source, target, store.CompareOptions{
		ExcludePatterns: excl,
		PageSize:        pageSize,
	})
	if err != nil {
		return fmt.Errorf("error during comparison: %w", err)
	}

	if len(diffs) > 0 {
		fmt.Printf("Database values differ. Tables: %s\n", strings.Join(diffs, ", "))
		os.Exit(1)
	}

	fmt.Println("Database values are same.")
	return nil
}

func versionCmdFn() string {
	version := "unknown"
	buildDate := "unkonwn"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				version = setting.Value
			} else if setting.Key == "vcs.time" {
				buildDate = setting.Value
			}
		}
	}
	version = fmt.Sprintf("\t: %s", version[:7])
	buildDate = fmt.Sprintf("build date\t: %s", buildDate)
	return strings.Join([]string{version, buildDate}, "\n")
}
