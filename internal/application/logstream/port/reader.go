package port

import "context"

// Reader keeps a source connection, but must not retain file handles that block appends.
type Reader interface {
	Read(ctx context.Context, offset int64, limit int) (data []byte, exists bool, err error)
	Close() error
}
