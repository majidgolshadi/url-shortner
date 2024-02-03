package cassandra

import (
	"context"
	"github.com/gocql/gocql"
	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/storage"
)

type row struct {
	Token string
	Url   string
}

type repository struct {
	db *gocql.Session
}

func NewRepository(db *gocql.Session) storage.Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) Save(ctx context.Context, url *domain.Url) error {
	return r.db.Query("INSERT INTO url_token(token, url) VALUES (?, ?)",
		url.Token, url.UrlPath).WithContext(ctx).Exec()
}

func (r *repository) Delete(ctx context.Context, token string) error {
	return r.db.Query("DELETE FROM url_token WHERE token = ?", token).WithContext(ctx).Exec()
}

func (r *repository) Fetch(ctx context.Context, token string) (*domain.Url, error) {
	var cassRow row
	err := r.db.Query("SELECT url FROM url_token WHERE token=?", token).WithContext(ctx).Scan(cassRow.Url)

	if err != nil {
		return nil, err
	}

	return &domain.Url{
		Token:   token,
		UrlPath: cassRow.Url,
	}, nil
}
