package dto

// InitializeInput contains one-time SSH credentials. These values are only
// used for the current host initialization and are never persisted.
type InitializeInput struct {
	Username             string
	Password             string
	PrivateKey           string
	PrivateKeyPassphrase string
}
