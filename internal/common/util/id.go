package repository

import (
	"github.com/oklog/ulid/v2"
)

func NewId() string {
	return ulid.Make().String()
}
