package mysql

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/storage"
)

type sqlRow struct {
	token string `db:"token"`
	url   string `db:"url"`
}
type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) storage.Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) Save(ctx context.Context, url *domain.Url) error {
	sql := `INSERT INTO url_token(token, url) VALUES(?, ?);`
	_, err := r.db.ExecContext(ctx, sql, url.Token, url.UrlPath)
	return err
}

func (r *repository) Delete(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM url_token WHERE token=?`, token)
	return err
}

func (r *repository) Fetch(ctx context.Context, token string) (*domain.Url, error) {
	row := sqlRow{}
	err := r.db.GetContext(ctx, &row, `SELECT token, url FROM url_token WHERE token = ?`, token)
	if err != nil {
		return nil, err
	}

	return &domain.Url{
		UrlPath: row.url,
		Token:   row.token,
	}, nil
}

func (r *repository) HealthCheck(ctx context.Context) (bool, interface{}) {
	err := r.db.PingContext(ctx)

	if err != nil {
		return false, struct {
			Status   bool
			ErrorMsg string
		}{
			Status:   false,
			ErrorMsg: err.Error(),
		}
	}

	return true, struct {
		Status bool
	}{
		Status: true,
	}
}
