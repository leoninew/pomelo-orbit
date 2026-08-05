package repository

import "errors"

// ErrNotFound reports that a repository singleton lookup found no matching record.
var ErrNotFound = errors.New("repository: not found")

// ErrReferenced reports that a resource cannot be removed while another record references it.
var ErrReferenced = errors.New("repository: referenced")
