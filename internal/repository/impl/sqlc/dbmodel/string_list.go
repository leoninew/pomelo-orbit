package dbmodel

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

func StringListJSON(values []string) (sql.NullString, error) {
	if values == nil {
		return sql.NullString{}, nil
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(encoded), Valid: true}, nil
}

func StringListFromJSON(value sql.NullString) ([]string, error) {
	if !value.Valid {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(value.String), &values); err != nil {
		return nil, err
	}
	if values == nil {
		return nil, fmt.Errorf("must be a JSON array")
	}
	return values, nil
}
