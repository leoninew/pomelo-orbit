package model

import "time"

// EnvironmentCredential is the SSH deployment key for an Environment.
type EnvironmentCredential struct {
	Id                  string    `db:"id"`
	ProjectId           string    `db:"project_id"`
	PublicKey           string    `db:"public_key"`
	EncryptedPrivateKey string    `db:"encrypted_private_key"`
	Revision            int64     `db:"revision"`
	CreatedAt           time.Time `db:"created_at"`
}
