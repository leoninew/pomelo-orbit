package tx

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/requestid"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func TestMiddlewareCommitOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openMemory(t)
	if _, err := db.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	r := newRouter(db)
	r.POST("/api/ok", func(c *gin.Context) {
		conn, err := DbTXFrom(c.Request.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.ExecContext(c.Request.Context(), `INSERT INTO probe(v) VALUES ('x')`); err != nil {
			t.Fatal(err)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/ok", nil)
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

	r := newRouter(db)
	r.POST("/api/fail", func(c *gin.Context) {
		conn, err := DbTXFrom(c.Request.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.ExecContext(c.Request.Context(), `INSERT INTO probe(v) VALUES ('x')`); err != nil {
			t.Fatal(err)
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "bad"})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/fail", nil)
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

func TestMiddlewareWritesUnifiedErrorWhenBeginFails(t *testing.T) {
	db, mock := openMock(t)
	mock.ExpectBegin().WillReturnError(errors.New("database password"))

	called := false
	r := newRouter(db)
	r.POST("/api/begin", func(c *gin.Context) {
		called = true
		c.Status(http.StatusCreated)
	})

	w := serve(t, r, http.MethodPost, "/api/begin")
	if called {
		t.Fatal("handler ran after transaction begin failure")
	}
	assertInternalError(t, w)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMiddlewareDoesNotOpenTransactionForReadRequest(t *testing.T) {
	db, mock := openMock(t)
	r := newRouter(db)
	r.GET("/api/read", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := serve(t, r, http.MethodGet, "/api/read")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMiddlewareDiscardsSuccessResponseWhenCommitFails(t *testing.T) {
	db, mock := openMock(t)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO probe").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("database password"))

	r := newRouter(db)
	r.POST("/api/commit", func(c *gin.Context) {
		connection, err := DbTXFrom(c.Request.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := connection.ExecContext(c.Request.Context(), "INSERT INTO probe(v) VALUES ('x')"); err != nil {
			t.Fatal(err)
		}
		c.JSON(http.StatusCreated, gin.H{"created": true})
	})

	w := serve(t, r, http.MethodPost, "/api/commit")
	assertInternalError(t, w)
	if body := w.Body.String(); body == "" || strings.Contains(body, "created") {
		t.Fatalf("commit failure leaked buffered success body: %q", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMiddlewarePanicDiscardsBufferedSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openMemory(t)
	if _, err := db.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	addRequestId(r)
	r.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", errors.New("handler panic")))
	}))
	r.Use(Middleware(db))
	r.POST("/api/panic", func(c *gin.Context) {
		connection, err := DbTXFrom(c.Request.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := connection.ExecContext(c.Request.Context(), `INSERT INTO probe(v) VALUES ('x')`); err != nil {
			t.Fatal(err)
		}
		c.JSON(http.StatusCreated, gin.H{"created": true})
		panic("handler panic")
	})

	w := serve(t, r, http.MethodPost, "/api/panic")
	assertInternalError(t, w)
	if body := w.Body.String(); body == "" || strings.Contains(body, "created") {
		t.Fatalf("panic leaked buffered success body: %q", body)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM probe`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected rollback after panic, got %d rows", count)
	}
}

func newRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	addRequestId(r)
	r.Use(Middleware(db))
	return r
}

func addRequestId(r *gin.Engine) {
	r.Use(func(c *gin.Context) {
		c.Set(requestid.ContextKey, "request-1")
		c.Header(requestid.HeaderName, "request-1")
		c.Next()
	})
}

func serve(t *testing.T, router *gin.Engine, method string, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertInternalError(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", recorder.Code)
	}
	if recorder.Header().Get(requestid.HeaderName) != "request-1" {
		t.Fatalf("request id header=%q", recorder.Header().Get(requestid.HeaderName))
	}
	var response transportresponse.ErrorResp
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response != (transportresponse.ErrorResp{
		Code: "internal_error", Error: "Internal server error.", RequestId: "request-1",
	}) {
		t.Fatalf("response=%#v", response)
	}
}

func openMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
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
