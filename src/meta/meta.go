package meta

import "github.com/jmoiron/sqlx"

type App interface {
	TabName() string

	Render() string

	SetupSchema(db *sqlx.DB) (int, error)
}
