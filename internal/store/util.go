package store

import (
	"strings"

	"github.com/go-sql-driver/mysql"
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
