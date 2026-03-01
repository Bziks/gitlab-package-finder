package migrate

import (
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/bziks/gitlab-package-finder/internal/app"
	"github.com/bziks/gitlab-package-finder/internal/config"
)

const (
	_argumentUp   = "up"
	_argumentDown = "down"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:       "migrate {up | down}",
		Short:     "Run migrations",
		ValidArgs: []string{_argumentUp, _argumentDown},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.New()
			if err != nil {
				return errors.Wrap(err, "load config")
			}

			db, err := app.InitMySQL(cfg.Mysql)
			if err != nil {
				return errors.Wrap(err, "init mysql")
			}
			defer db.Close()

			driver, err := mysql.WithInstance(
				db.DB,
				&mysql.Config{
					DatabaseName: cfg.Mysql.DbName,
				},
			)
			if err != nil {
				return errors.Wrap(err, "can't create driver from mysql instance")
			}

			m, err := migrate.NewWithDatabaseInstance(
				fmt.Sprintf("file://%s", cfg.Mysql.MigrationsPath),
				"mysql",
				driver,
			)
			if err != nil {
				return errors.Wrap(err, "can't init migration from driver")
			}

			switch args[0] {
			case _argumentUp:
				err = m.Up()
			case _argumentDown:
				err = m.Down()
			}

			if err != nil && !errors.Is(err, migrate.ErrNoChange) {
				return errors.Wrap(err, "can't migrate")
			}

			slog.Info("Migration completed")

			return nil
		},
	}
}
