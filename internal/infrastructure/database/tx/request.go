package tx

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Middleware begins a request-scoped transaction and commits only when the
// handler finishes without abort and without recording errors.
// Validation/business 4xx still roll back to avoid partial writes.
func Middleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := WithDB(c.Request.Context(), db)
		sqlTx, err := db.BeginTx(ctx, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "failed to begin transaction"})
			return
		}
		ctx = WithTx(ctx, sqlTx)
		c.Request = c.Request.WithContext(ctx)

		committed := false
		defer func() {
			if !committed {
				_ = sqlTx.Rollback()
			}
		}()

		c.Next()

		if c.IsAborted() || len(c.Errors) > 0 || c.Writer.Status() >= http.StatusBadRequest {
			return
		}
		if err := sqlTx.Commit(); err != nil {
			_ = c.Error(err)
			return
		}
		committed = true
	}
}
