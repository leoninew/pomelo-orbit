package model

import "time"

type Credential struct {
	Id            string    `db:"id"`
	ProjectId     *string   `db:"project_id"`
	Name          string    `db:"name"`
	Type          string    `db:"type"`
	EncryptedData string    `db:"encrypted_data"`
	CreatedAt     time.Time `db:"created_at"`
}
