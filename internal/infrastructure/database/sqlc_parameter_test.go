package db

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLCStatementsDoNotMixNamedAndAnonymousParameters(t *testing.T) {
	queryRoot := filepath.Join(repositoryRoot(t), "sql", "query")
	err := filepath.WalkDir(queryRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".sql" {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, statement := range strings.Split(string(contents), "-- name:")[1:] {
			if !strings.Contains(statement, "sqlc.arg(") && !strings.Contains(statement, "sqlc.narg(") {
				continue
			}
			if hasAnonymousParameter(statement) {
				name := strings.TrimSpace(strings.SplitN(statement, "\n", 2)[0])
				t.Errorf("%s statement %s mixes SQLC named and anonymous parameters", path, name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "sqlc.yaml")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("repository root containing sqlc.yaml was not found")
		}
		root = parent
	}
}

func hasAnonymousParameter(statement string) bool {
	inString := false
	for index := 0; index < len(statement); index++ {
		if statement[index] == '\'' {
			if inString && index+1 < len(statement) && statement[index+1] == '\'' {
				index++
				continue
			}
			inString = !inString
			continue
		}
		if !inString && statement[index] == '?' && (index+1 == len(statement) || statement[index+1] < '0' || statement[index+1] > '9') {
			return true
		}
	}
	return false
}
