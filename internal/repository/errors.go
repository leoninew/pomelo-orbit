package repository

import "errors"

// ErrNotFound reports that a repository singleton lookup found no matching record.
var ErrNotFound = errors.New("repository: not found")
