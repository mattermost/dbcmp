package store

import (
	"database/sql/driver"
	"encoding/base32"
	"encoding/json"
	"errors"

	"github.com/pborman/uuid"
)

var (
	encoding = base32.NewEncoding("ybndrfg8ejkmcpqxot1uwisza345h769").WithPadding(base32.NoPadding)
)

type testStruct1 struct {
	Id          string `db:"Id"`
	CreateAt    int64  `db:"CreateAt"`
	Name        string `db:"Name"`
	Description string `db:"Description"`
}

type testStruct2 struct {
	Id        string          `db:"Id"`
	AnotherId string          `db:"AnotherId"`
	IsActive  bool            `db:"IsActive"`
	Props     stringInterface `db:"Props"`
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

	return string(j), err
}

// newId creates a unique identifier like being done in MM
func newId() string {
	return encoding.EncodeToString(uuid.NewRandom())
}
