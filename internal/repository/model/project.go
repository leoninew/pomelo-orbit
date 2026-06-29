package model

import "time"

type Project struct {
	Id        string    `db:"id"`
	Name      string    `db:"name"`
	Code      string    `db:"code"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ProjectMember struct {
	ProjectId string    `db:"project_id"`
	UserId    string    `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
}
