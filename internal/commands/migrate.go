package commands

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/eugenetriguba/bolt/internal/configloader"
	"github.com/eugenetriguba/bolt/internal/repositories"
	"github.com/eugenetriguba/bolt/internal/services"
	"github.com/eugenetriguba/bolt/internal/storage"
	"github.com/google/subcommands"
)

type UpgradeCmd struct{}

func (*UpgradeCmd) Name() string { return "upgrade" }

func (*UpgradeCmd) Synopsis() string { return "Upgrade the database to the latest migration." }
func (*UpgradeCmd) Usage() string {
	return `upgrade:
	Upgrade the database to the latest migration.
  `
}

func (m *UpgradeCmd) SetFlags(f *flag.FlagSet) {}

func (m *UpgradeCmd) Execute(
	_ context.Context,
	f *flag.FlagSet,
	_ ...interface{},
) subcommands.ExitStatus {
	cfg, err := configloader.NewConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	db, err := storage.DBConnect(
		cfg.Connection.Driver,
		storage.DBConnectionString(&cfg.Connection),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}
	defer db.Close()

	migrationDBRepo, err := repositories.NewMigrationDBRepo(db)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	migrationFsRepo, err := repositories.NewMigrationFsRepo(cfg.MigrationsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	migrationService := services.NewMigrationService(migrationDBRepo, migrationFsRepo)
	err = migrationService.ApplyAllMigrations()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
