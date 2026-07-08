package repository

import "github.com/jmoiron/sqlx"

type Store struct {
	db     *sqlx.DB
	driver string
}

func NewStore(db *sqlx.DB, driver string) Store {
	return Store{db: db, driver: driver}
}

func (s Store) DB() *sqlx.DB {
	return s.db
}

func (s Store) Driver() string {
	return s.driver
}
