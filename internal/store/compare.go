package store

import (
	"fmt"
	"strings"
)

type CompareOptions struct {
	ExcludePatterns []string
	Verbose         bool
	PageSize        int
}

func Compare(srcDSN, dstDSN string, opts CompareOptions) ([]string, error) {
	srcdb, err := NewDB(srcDSN)
	if err != nil {
		return nil, fmt.Errorf("could not initiate src db connection: %w", err)
	}
	defer srcdb.sqlDB.Close()

	srcTables, err := srcdb.TableList()
	if err != nil {
		return nil, fmt.Errorf("could not list src tables: %w", err)
	}

	dstdb, err := NewDB(dstDSN)
	if err != nil {
		return nil, fmt.Errorf("could not initiate dst db connection: %w", err)
	}
	defer dstdb.sqlDB.Close()

	dstTables, err := dstdb.TableList()
	if err != nil {
		return nil, fmt.Errorf("could not list dst tables: %w", err)
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

tableLoop:
	for k, v := range srcTables {
		v2, ok := dstTables[strings.ToLower(k)]
		if !ok {
			return nil, fmt.Errorf("%q table is not found in dst schema", k)
		}

		// we do a count comparison to save some resources before diving deeper
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

		remainig := opts.PageSize
		var cd1, cd2 cursorData
		var srcCheksum, dstChecksum string

		// loop until no remaining rows left to calculate checksum
		for remainig > 0 {
			if cd1.cursors == nil {
				cd1.limit = remainig
			}
			if cd2.cursors == nil {
				cd2.limit = remainig
			}
			srcCheksum, cd1, err = srcdb.checksum(v, cd1)
			if err != nil {
				return nil, fmt.Errorf("could not compute src checksum: %w", err)
			}

			dstChecksum, cd2, err = dstdb.checksum(v2, cd2)
			if err != nil {
				return nil, fmt.Errorf("could not compute dst checksum: %w", err)
			}

			if srcCheksum != dstChecksum {
				mismatchs = append(mismatchs, v.TableName)
				continue tableLoop
			}

			if cd1.limit != cd2.limit {
				return nil, fmt.Errorf("could not compute checksum: cursors are out of sync")
			}

			remainig = cd1.limit
		}

	}

	return mismatchs, nil
}
