package dbstore

import (
	"embed"

	"github.com/Boostport/migration"
	"github.com/Boostport/migration/driver/postgres"
)

//go:embed migrations
var embedFS embed.FS

func MigrationSource() *migration.EmbedMigrationSource {
	return &migration.EmbedMigrationSource{
		EmbedFS: embedFS,
		Dir:     "migrations",
	}
}

func RunMigration(dsn string) (int, error) {
	driver, err := postgres.New(dsn)

	if err != nil {
		return 0, err
	}

	applied, err := migration.Migrate(driver, MigrationSource(), migration.Up, 0)

	return applied, err
}
