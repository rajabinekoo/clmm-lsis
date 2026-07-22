package postgres

import (
	"database/sql"
	"fmt"

	"github.com/rajabinekoo/clmm-lsis/internal/reconstruction"
)

const defaultSchema = "public"

// Repository reads the legacy research tables and the future append-only swap
// table.
//
// Existing tables are treated as read-only.
type Repository struct {
	db     *sql.DB
	schema string
}

func NewRepository(
	db *sql.DB,
) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf(
			"create PostgreSQL repository: database is nil",
		)
	}

	return &Repository{
		db:     db,
		schema: defaultSchema,
	}, nil
}

var _ reconstruction.HistoricalSource = (*Repository)(nil)
