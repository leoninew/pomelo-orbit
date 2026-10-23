package bootstrap

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	taskrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/task"
)

func TestHTTPTransactionAssemblyExcludesFileAndCertificateOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	deps := newHTTPServerDependencies(config.Config{Runtime: &config.Runtime{}}, slog.New(slog.NewTextHandler(io.Discard, nil)), database, taskrepo.NewRepository(database))
	for _, test := range []struct {
		method, pattern, path string
		transaction           bool
	}{
		{http.MethodPut, "/api/setting/config", "/api/setting/config", false},
		{http.MethodDelete, "/api/setting/config", "/api/setting/config", false},
		{http.MethodPost, "/api/route/:route_id/mkcert", "/api/route/route-1/mkcert", false},
		{http.MethodPost, "/api/application", "/api/application", true},
		{http.MethodPost, "/api/auth/google/callback", "/api/auth/google/callback", false},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			if test.transaction {
				mock.ExpectBegin()
				mock.ExpectCommit()
			}
			router := gin.New()
			router.Use(deps.MutatingUnitOfWork)
			router.Handle(test.method, test.pattern, func(c *gin.Context) {
				_, hasTransaction := tx.TxFrom(c.Request.Context())
				if hasTransaction != test.transaction {
					t.Errorf("transaction=%v", hasTransaction)
				}
				c.Status(http.StatusNoContent)
			})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != http.StatusNoContent {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
