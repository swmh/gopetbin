package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/swmh/gopetbin/internal/service"
	"github.com/swmh/gopetbin/pkg/retry"
)

type DB struct {
	db *sqlx.DB
}

type Paste struct {
	ID             string        `db:"id"`
	Name           string        `db:"name"`
	ExpireAt       time.Time     `db:"expire_at"`
	RemainingReads sql.NullInt64 `db:"remaining_reads"`
}

func New(address, user, password, dbname string) (*DB, error) {
	db, err := sqlx.Open("pgx", fmt.Sprintf("postgres://%s:%s@%s/%s", user, password, address, dbname))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = retry.Retry(ctx, func(ctx context.Context) error {
		return db.PingContext(ctx)
	})

	if err != nil {
		return nil, fmt.Errorf("cannot connect to db: %w", err)
	}

	return &DB{db}, nil
}

func (d *DB) IsNoSuchPaste(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func (d *DB) CreatePaste(ctx context.Context, id string, name string, expire time.Time, burn int) error {
	var remainingReads sql.NullInt64
	if burn > 0 {
		remainingReads = sql.NullInt64{
			Int64: int64(burn),
			Valid: true,
		}
	}

	_, err := d.db.ExecContext(ctx, `INSERT INTO pastes (id, name, expire_at, remaining_reads)
									VALUES ($1, $2, $3, $4)`, id, name, expire, remainingReads)
	return err
}

func (d *DB) GetPaste(ctx context.Context, id string) (service.Paste, error) {
	var paste Paste

	err := d.db.QueryRowxContext(ctx, "SELECT id, name, expire_at, remaining_reads FROM pastes WHERE id = $1", id).StructScan(&paste)
	if err != nil {
		return service.Paste{}, err
	}

	var burnAfter int
	var IsBurnable bool

	if paste.RemainingReads.Valid {
		burnAfter = int(paste.RemainingReads.Int64)
		IsBurnable = true
	}

	return service.Paste{
		Name:       paste.Name,
		Expire:     paste.ExpireAt,
		BurnAfter:  burnAfter,
		IsBurnable: IsBurnable,
	}, nil
}

func (d *DB) Readed(ctx context.Context, id string) error {
	_, err := d.db.ExecContext(ctx, `UPDATE pastes SET remaining_reads = remaining_reads - 1 WHERE id = $1`, id)
	return err
}

func (d *DB) GetExpired(ctx context.Context) ([]string, error) {
	r, err := d.db.QueryxContext(ctx,
		`SELECT name FROM pastes GROUP BY name
		HAVING COUNT(*) = SUM(CASE WHEN remaining_reads = 0 OR expire_at < NOW() THEN 1 ELSE 0 END)`,
	)
	if err != nil {
		return nil, err
	}

	var pastes []string

	for r.Next() {
		var v string

		err = r.Scan(&v)
		if err != nil {
			return nil, err
		}

		pastes = append(pastes, v)
	}

	return pastes, nil
}
