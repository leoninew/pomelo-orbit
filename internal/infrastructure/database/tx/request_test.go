package tx

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func TestMiddlewareCommitOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openMemory(t)
	if _, err := db.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(Middleware(db))
	r.POST("/ok", func(c *gin.Context) {
		conn, err := DBTXFrom(c.Request.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.ExecContext(c.Request.Context(), `INSERT INTO probe(v) VALUES ('x')`); err != nil {
			t.Fatal(err)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM probe`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected committed row, got %d", count)
	}
}

func TestMiddlewareRollbackOnError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openMemory(t)
	if _, err := db.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(Middleware(db))
	r.POST("/fail", func(c *gin.Context) {
		conn, err := DBTXFrom(c.Request.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.ExecContext(c.Request.Context(), `INSERT INTO probe(v) VALUES ('x')`); err != nil {
			t.Fatal(err)
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "bad"})
	})

	req := httptest.NewRequest(http.MethodPost, "/fail", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM probe`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected rollback, got %d rows", count)
	}
}

func openMemory(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
