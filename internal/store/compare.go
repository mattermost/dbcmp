package store

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

type CompareOptions struct {
	ExcludePatterns []string
	IncludePatterns []string
	Verbose         bool
	PageSize        int
	FailFast        bool
	SkipCount       bool
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

	mismatchs := make(map[string]any)

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

		var c1, c2 int
		if !opts.SkipCount {
			// we do a count comparison to save some resources before diving deeper
			c1, err = srcdb.count(v)
			if err != nil {
				return nil, fmt.Errorf("could not count rows of %q: %w", v.TableName, err)
			}
			c2, err = dstdb.count(v2)
			if err != nil {
				return nil, fmt.Errorf("could not count rows of %q: %w", v2.TableName, err)
			}
			if c1 != c2 {
				fmt.Printf("number of rows did not match for %q(%d, %d)\n", v.TableName, c1, c2)
				mismatchs[v.TableName] = struct{}{}
				continue
			} else if c1 == 0 {
				continue
			}
		}

		remaining := opts.PageSize
		var cd1, cd2 cursorData
		var srcChecksum, dstChecksum string

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

			var errSrc, errDst error
			wg := sync.WaitGroup{}

			wg.Add(1)
			go func() {
				defer wg.Done()
				srcChecksum, cd1, errSrc = srcdb.checksum(v, cd1)
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				dstChecksum, cd2, errDst = dstdb.checksum(v2, cd2)
			}()
			wg.Wait()

			if errSrc != nil && cd1.limit == 0 {
				break
			} else if errSrc != nil {
				return nil, fmt.Errorf("could not compute src checksums: %w", errSrc)
			}

			if errDst != nil && cd2.limit == 0 {
				break
			} else if errDst != nil {
				return nil, fmt.Errorf("could not compute dst checksums: %w", errDst)
			}

			if srcChecksum != dstChecksum {
				mismatchs[v.TableName] = struct{}{}

				// if verbose flag is set, we print the diff by scanning rows one by one
				if opts.Verbose {
				rowLoop:
					for i := 0; i < remaining; i++ {
						// we still need to use checksum methodology to have
						// consistency on the comparison.
						wg2 := sync.WaitGroup{}

						wg2.Add(1)
						go func() {
							defer wg2.Done()
							srcChecksum, prevCd1, errSrc = srcdb.checksum(v, cursorData{
								cursors: slices.Clone(prevCd1.cursors),
								limit:   1,
							})
						}()

						wg2.Add(1)
						go func() {
							defer wg2.Done()
							dstChecksum, prevCd2, errDst = dstdb.checksum(v2, cursorData{
								cursors: slices.Clone(prevCd2.cursors),
								limit:   1,
							})
						}()
						wg2.Wait()

						if errSrc != nil && prevCd1.limit == 0 {
							fmt.Println("reached end of the batch")
							break rowLoop
						} else if errSrc != nil {
							return nil, fmt.Errorf("could not compute src checksum on single row: %w", errSrc)
						}

						if errDst != nil && prevCd2.limit == 0 {
							fmt.Println("reached end of the batch")
							break rowLoop
						} else if errDst != nil {
							return nil, fmt.Errorf("could not compute dst checksum on single row: %w", errDst)
						}

						if srcChecksum == dstChecksum {
							continue rowLoop
						}

						// we only print the primary keys if two rows differ
						t := "Row differs: "
						for i := range prevCd1.cursors {
							t += fmt.Sprintf("%s: %s ", v.PrimaryKeys[i], prevCd1.cursors[i])
						}
						fmt.Println(strings.TrimSpace(t))

						if opts.FailFast {
							break rowLoop
						}
					}
				}

				if opts.FailFast {
					break tableLoop
				}
			}

			if cd1.limit != cd2.limit {
				return nil, fmt.Errorf("could not compute checksum: cursors are out of sync")
			}

			remaining = cd1.limit

			elapsed := time.Since(start)
			// report progress every minute
			if elapsed > time.Minute {
				fmt.Printf("Number of rows processed: %d\n", opts.PageSize*loopCount)
				start = time.Now() // reset the start time
			}
		}
	}

	tables := make([]string, 0, len(mismatchs))
	for table := range mismatchs {
		tables = append(tables, table)
	}

	return tables, nil
}
