package model

import "time"

// Environment is a project-scoped deployment target.
type Environment struct {
	Id          string    `db:"id"`
	ProjectId   string    `db:"project_id"`
	Code        string    `db:"code"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
