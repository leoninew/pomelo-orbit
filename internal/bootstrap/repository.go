package bootstrap

import (
	"github.com/jmoiron/sqlx"

	store "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
)

func NewRepositoryStore(database *sqlx.DB, driver string) store.Store {
	return store.NewStore(database, driver)
}

func NewTaskRepository(database *sqlx.DB, driver string) taskrepo.Repository {
	return taskrepo.NewRepository(database, driver)
}
