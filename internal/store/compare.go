package store

import (
	"fmt"
	"strings"
)

type CompareOptions struct {
	ExcludePatterns []string
	Verbose         bool
}

func Compare(srcDSN, dstDSN string, opts CompareOptions) ([]string, error) {
	srcdb, err := NewDB(srcDSN)
	if err != nil {
		return nil, fmt.Errorf("could not initiate src db connection: %w", err)
	}
	defer srcdb.sqlDB.Close()

	srcTables, err := srcdb.TableList()
	if err != nil {
		return nil, fmt.Errorf("could not list tables")
	}

	dstdb, err := NewDB(dstDSN)
	if err != nil {
		return nil, fmt.Errorf("could not initiate dst db connection: %w", err)
	}
	defer dstdb.sqlDB.Close()

	dstTables, err := dstdb.TableList()
	if err != nil {
		return nil, fmt.Errorf("could not list tables")
	}

	excl := sliceToMap(opts.ExcludePatterns)

	// find a more elegant solution fo this
	// essentially we want to exclude some
	// patterns from comparing.
	for k := range srcTables {
		for e := range excl {
			if strings.Contains(k, strings.ToLower(e)) {
				delete(srcTables, k)
			}
		}
	}

	var mismatchs []string
	for k, v := range srcTables {
		v2, ok := dstTables[strings.ToLower(k)]
		if !ok {
			return nil, fmt.Errorf("%q table is not found in dst schema", k)
		}

		c1, err := srcdb.count(v)
		if err != nil {
			return nil, fmt.Errorf("could not count rows of %q: %w", v.TableName, err)
		}
		c2, err := dstdb.count(v2)
		if err != nil {
			return nil, fmt.Errorf("could not count rows of %q: %w", v2.TableName, err)
		}
		if c1 != c2 {
			mismatchs = append(mismatchs, v.TableName)
			continue
		} else if c1 == 0 {
			continue
		}

		srcCheksum, err := srcdb.checksum(v)
		if err != nil {
			return nil, fmt.Errorf("could not compute checksum: %w", err)
		}

		dstChecksum, err := dstdb.checksum(v2)
		if err != nil {
			return nil, fmt.Errorf("could not compute checksum: %w", err)
		}

		if srcCheksum != dstChecksum {
			mismatchs = append(mismatchs, v.TableName)
			continue
		}
	}

	return mismatchs, nil
}
