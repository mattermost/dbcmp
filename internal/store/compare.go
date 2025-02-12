package store

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type CompareOptions struct {
	ExcludePatterns []string
	IncludePatterns []string
	Verbose         bool
	PageSize        int
	FailFast        bool
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
	incl := sliceToMap(opts.IncludePatterns)

	if len(incl) > 0 && len(excl) > 0 {
		return nil, fmt.Errorf("include and exclude flags cannot be used together")
	}

	if len(incl) > 0 {
		// include removes elements from the input map if they are not included.
		// works with exact match and case-insensitive.
		srcTables = filterMap(srcTables, opts.IncludePatterns, include)
	} else if len(excl) > 0 {
		// exclude removes elements from the input map by the given keys.
		// filteration is case-insensitive and made with strings.Contains.
		srcTables = filterMap(srcTables, opts.ExcludePatterns, exclude)
	}

	var mismatchs []string

	fmt.Printf("%d table(s) going to be compared. ", len(srcTables))
	tableNames := make([]string, 0, len(srcTables))
	for _, v := range srcTables {
		tableNames = append(tableNames, v.TableName)
	}
	fmt.Printf("{%q}\n", strings.Join(tableNames, ", "))

tableLoop:
	for k, v := range srcTables {
		fmt.Printf("Comparing %q(%s) with %q(%s)\n", v.TableName, srcdb.dbType, k, dstdb.dbType)
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

		remaining := opts.PageSize
		var cd1, cd2 cursorData
		var srcCheksum, dstChecksum string

		start := time.Now()
		var loopCount int
		// loop until no remaining rows left to calculate checksum
		for remaining > 0 {
			loopCount++
			if cd1.cursors == nil {
				cd1.limit = remaining
			}
			if cd2.cursors == nil {
				cd2.limit = remaining
			}
			// we need to save the previous cursors as they are automatically updated
			// by the checksum query.
			prevCd1 := cursorData{
				cursors: cd1.cursors,
			}
			prevCd2 := cursorData{
				cursors: cd2.cursors,
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

				// if verbose flag is set, we print the diff by scanning rows one by one
				if opts.Verbose {
				rowLoop:
					for i := 0; i < remaining; i++ {
						// we still need to use checksum methodology to have
						// consistency on the comparison.
						srcCheksum, prevCd1, err = srcdb.checksum(v, cursorData{
							cursors: prevCd1.cursors,
							limit:   1,
						})
						if err != nil {
							return nil, fmt.Errorf("could not compute src checksum: %w", err)
						}

						dstChecksum, prevCd2, err = dstdb.checksum(v2, cursorData{
							cursors: prevCd2.cursors,
							limit:   1,
						})
						if err != nil {
							return nil, fmt.Errorf("could not compute dst checksum: %w", err)
						}

						if srcCheksum == dstChecksum {
							continue
						}

						srcRow, err := srcdb.GetRow(v, prevCd1, i)
						if err != nil {
							return nil, fmt.Errorf("could not get src row: %w", err)
						}

						dstRow, err := dstdb.GetRow(v2, prevCd2, i)
						if err != nil {
							return nil, fmt.Errorf("could not get dst row: %w", err)
						}

						var pkDiffer bool
						for _, pk := range v.PrimaryKeys {
							if srcRow[pk] != dstRow[pk] {
								pkDiffer = true
							}
						}

						// there is a chance that the primary keys are different
						// which means the rows are different, this is rare but possible
						if pkDiffer {
							t := "Row differs in primary keys: "
							for _, pk := range v.PrimaryKeys {
								t += fmt.Sprintf("%s: src=%v dst=%v ", pk, srcRow[strings.ToLower(pk)], dstRow[strings.ToLower(pk)])
							}
							fmt.Println(strings.TrimSpace(t))
						} else {
							// we only print the primary keys if two rows differ
							t := "Row differs: "
							for _, pk := range v.PrimaryKeys {
								t += fmt.Sprintf("%s: src=%v dst=%v ", pk, srcRow[strings.ToLower(pk)], dstRow[strings.ToLower(pk)])
							}
							fmt.Println(strings.TrimSpace(t))
						}

						if opts.FailFast {
							break rowLoop
						}
					}
				}

				continue tableLoop
			}

			if cd1.limit != cd2.limit {
				return nil, fmt.Errorf("could not compute checksum: cursors are out of sync")
			}

			remaining = cd1.limit

			elapsed := time.Since(start)
			// report progress every minute
			if elapsed > time.Minute {
				fmt.Printf("Table progress: %.0f%%\n", math.Round(float64(opts.PageSize*loopCount)/float64(c1))*100)
				start = time.Now() // reset the start time
			}

			if opts.FailFast {
				break tableLoop
			}
		}

	}

	return mismatchs, nil
}
