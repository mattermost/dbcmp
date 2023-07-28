package store

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type testStruct1 struct {
	Id          string `db:"Id"`
	CreateAt    int64  `db:"CreateAt"`
	Name        string `db:"Name"`
	Description string `db:"Description"`
}

type testStruct2 struct {
	Id          string
	IsActive    bool
	Props       stringInterface
	Description string
}

type stringInterface map[string]any

func (si *stringInterface) Scan(value any) error {
	if value == nil {
		return nil
	}

	buf, ok := value.([]byte)
	if ok {
		return json.Unmarshal(buf, si)
	}

	str, ok := value.(string)
	if ok {
		return json.Unmarshal([]byte(str), si)
	}

	return errors.New("received value is neither a byte slice nor string")
}

// Value converts StringInterface to database value
func (si stringInterface) Value() (driver.Value, error) {
	j, err := json.Marshal(si)
	if err != nil {
		return nil, err
	}

	// non utf8 characters are not supported https://mattermost.atlassian.net/browse/MM-41066
	return string(j), err
}
