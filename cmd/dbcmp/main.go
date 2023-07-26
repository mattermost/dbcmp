package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/isacikgoz/dbcmp/internal/store"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Minimum 2 arguments are required")
		os.Exit(1)
	}

	src := os.Args[1]
	dst := os.Args[2]

	diffs, err := store.Compare(src, dst, store.CompareOptions{
		ExcludePatterns: []string{
			"db_migrations",
			"db_lock",
			"ir_",
			"focalboard",
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during comparison: %s\n", err)
		os.Exit(1)
	}

	if len(diffs) == 0 {
		fmt.Println("Database values are same.")
		os.Exit(0)
	}

	fmt.Printf("Database values differ. Tables: %s\n", strings.Join(diffs, ", "))
}
