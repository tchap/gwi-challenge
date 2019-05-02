package migrations

import (
	"github.com/tchap/gwi-challenge/cmd/gwi-api/api/stores/postgrestore/migrations/sql"

	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/pkg/errors"
)

// NewSource returns a new migration source that can be used to perform DB migrations.
func NewSource() (source.Driver, error) {
	s := bindata.Resource(sql.AssetNames(),
		func(name string) ([]byte, error) {
			return sql.Asset(name)
		})

	driver, err := bindata.WithInstance(s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize migrations source driver")
	}
	return driver, nil
}
