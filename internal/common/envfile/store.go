package envfile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"time"
)

var ErrBusy = errors.New("configuration file is busy")

type Store struct {
	path      string
	directory string
}

func NewStore(path string) Store {
	return Store{path: filepath.Clean(path), directory: filepath.Join(filepath.Dir(path), "data", "config")}
}

func (s Store) Load(ctx context.Context) (map[string]string, error) {
	var values map[string]string
	err := s.withLock(ctx, func() error {
		content, err := s.read()
		if err != nil {
			return err
		}
		values, err = Decode(content)
		return err
	})
	return values, err
}

func (s Store) Mutate(ctx context.Context, change func(map[string]string) error) (map[string]string, error) {
	var result map[string]string
	err := s.withLock(ctx, func() error {
		original, err := s.read()
		if err != nil {
			return err
		}
		values, err := Decode(original)
		if err != nil {
			return err
		}
		if err := change(values); err != nil {
			return err
		}
		content, err := Encode(values)
		if err != nil {
			return err
		}
		// Reject assignments exceeding the parser's line limit before writing.
		if _, err := Decode(content); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !bytes.Equal(original, content) {
			if err := s.write(original, content); err != nil {
				return err
			}
		}
		result = maps.Clone(values)
		return nil
	})
	return result, err
}

func (s Store) withLock(ctx context.Context, operation func() error) error {
	if err := os.MkdirAll(s.directory, 0o700); err != nil {
		return fmt.Errorf("create configuration recovery directory: %w", err)
	}
	lock, err := os.OpenFile(s.record("lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open configuration lock: %w", err)
	}
	defer func() { _ = lock.Close() }()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		locked, err := tryLock(lock)
		if err != nil {
			return fmt.Errorf("lock configuration file: %w", err)
		}
		if locked {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return ErrBusy
		case <-ticker.C:
		}
	}
	defer unlock(lock)
	if err := s.recover(); err != nil {
		return err
	}
	return operation()
}

func (s Store) record(suffix string) string {
	return filepath.Join(s.directory, filepath.Base(s.path)+"."+suffix)
}

func (s Store) read() ([]byte, error) {
	content, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return content, err
}

func (s Store) recover() error {
	previous, err := os.ReadFile(s.record("prepared"))
	if err == nil {
		if err := writeInPlace(s.path, previous); err != nil {
			return fmt.Errorf("restore interrupted configuration write: %w", err)
		}
		if err := os.Remove(s.record("prepared")); err != nil {
			return fmt.Errorf("remove recovered configuration record: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read configuration recovery record: %w", err)
	}
	if err := os.Remove(s.record("committed")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove committed configuration record: %w", err)
	}
	return nil
}

func (s Store) write(previous, content []byte) error {
	staged, err := os.CreateTemp(s.directory, ".overwrite-*.env")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(staged.Name()) }()
	writeErr := writeAndSync(staged, previous)
	closeErr := staged.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return fmt.Errorf("prepare configuration recovery record: %w", err)
	}
	if err := os.Rename(staged.Name(), s.record("prepared")); err != nil {
		return fmt.Errorf("publish configuration recovery record: %w", err)
	}
	// The mounted target keeps its inode; only recovery records are renamed.
	if err := writeInPlace(s.path, content); err != nil {
		return errors.Join(fmt.Errorf("write configuration file: %w", err), s.recover())
	}
	if err := os.Rename(s.record("prepared"), s.record("committed")); err != nil {
		return errors.Join(fmt.Errorf("commit configuration file: %w", err), s.recover())
	}
	// A leftover committed record is safe to clean up on the next operation.
	_ = os.Remove(s.record("committed"))
	return nil
}

func writeInPlace(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	writeErr := writeAndSync(file, content)
	return errors.Join(writeErr, file.Close())
}

func writeAndSync(file *os.File, content []byte) error {
	if _, err := file.WriteAt(content, 0); err != nil {
		return err
	}
	if err := file.Truncate(int64(len(content))); err != nil {
		return err
	}
	return file.Sync()
}
