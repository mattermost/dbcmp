package store

import (
	"maps"
	"strings"

	"github.com/go-sql-driver/mysql"
)

type filterMode int

const (
	include filterMode = iota
	exclude
)

func normalizeDSN(dataSource string) (string, error) {
	if strings.HasPrefix(dataSource, "postgres") {
		return dataSource, nil
	}

	config, err := mysql.ParseDSN(dataSource)
	if err != nil {
		return "", err
	}

	if config.Params == nil {
		config.Params = map[string]string{}
	}

	config.Params["multiStatements"] = "true"
	return config.FormatDSN(), nil
}

func sliceToMap(inc []string) map[string]struct{} {
	m := make(map[string]struct{})
	for i := 0; i < len(inc); i++ {
		m[inc[i]] = struct{}{}
	}

	return m
}

func filterMap[V any](m map[string]V, keys []string, mode filterMode) map[string]V {
	result := maps.Clone(m)

	keySet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		keySet[strings.ToLower(k)] = struct{}{}
	}

	switch mode {
	case include:
		maps.DeleteFunc(result, func(k string, _ V) bool {
			_, exists := keySet[strings.ToLower(k)]
			return !exists
		})
	case exclude:
		for k := range result {
			for e := range keySet {
				if strings.Contains(k, strings.ToLower(e)) {
					delete(result, k)
				}
			}
		}
	default:
		panic("unknown filter mode")
	}

	return result
}
