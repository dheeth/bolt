package repositories_test

import (
	"testing"
	"time"

	"github.com/eugenetriguba/bolt/internal/bolttest"
	"github.com/eugenetriguba/bolt/internal/models"
	"github.com/eugenetriguba/bolt/internal/repositories"
	"github.com/eugenetriguba/checkmate/assert"
	"github.com/upper/db/v4"
)

func TestNewMigrationDBRepo_CreatesTable(t *testing.T) {
	testdb := bolttest.NewTestDB(t)
	exists, err := testdb.Session.Collection("bolt_migrations").Exists()
	assert.ErrorIs(t, err, db.ErrCollectionDoesNotExist)
	assert.False(t, exists)

	_, err = repositories.NewMigrationDBRepo(testdb)
	assert.Nil(t, err)

	exists, err = testdb.Session.Collection("bolt_migrations").Exists()
	assert.Nil(t, err)
	assert.True(t, exists)
}

func TestNewMigrationDBRepo_TableAlreadyExists(t *testing.T) {
	testdb := bolttest.NewTestDB(t)
	_, err := testdb.Session.SQL().Exec(`CREATE TABLE bolt_migrations(id INT NOT NULL PRIMARY KEY)`)
	assert.Nil(t, err)
	_, err = testdb.Session.SQL().Exec(`INSERT INTO bolt_migrations(id) VALUES (1);`)
	assert.Nil(t, err)

	_, err = repositories.NewMigrationDBRepo(testdb)
	assert.Nil(t, err)

	count, err := testdb.Session.Collection("bolt_migrations").Find().Count()
	assert.Nil(t, err)
	assert.Equal(t, count, uint64(1))
	row, err := testdb.Session.SQL().Select("id").From("bolt_migrations").QueryRow()
	assert.Nil(t, err)
	var scanResult int
	err = row.Scan(&scanResult)
	assert.Nil(t, err)
	assert.Equal(t, scanResult, 1)
}

func TestList_EmptyTable(t *testing.T) {
	db := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(db)
	assert.Nil(t, err)

	migrations, err := repo.List()
	assert.Nil(t, err)
	assert.Equal(t, len(migrations), 0)
}

func TestList_SingleResult(t *testing.T) {
	db := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(db)
	assert.Nil(t, err)

	version := "20230101000000"
	_, err = db.Session.SQL().InsertInto("bolt_migrations").Columns("version").Values(version).Exec()
	assert.Nil(t, err)

	migrations, err := repo.List()
	assert.Nil(t, err)
	assert.Equal(t, len(migrations), 1)
	assert.DeepEqual(
		t,
		migrations[version],
		&models.Migration{Version: version, Message: "", Applied: true},
	)
}

func TestList_ShortVersion(t *testing.T) {
	db := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(db)
	assert.Nil(t, err)

	version := "20230101"
	_, err = db.Session.SQL().InsertInto("bolt_migrations").Columns("version").Values(version).Exec()
	assert.Nil(t, err)

	migrations, err := repo.List()
	assert.Nil(t, err)
	assert.Equal(t, len(migrations), 1)
	assert.DeepEqual(
		t,
		migrations[version],
		&models.Migration{Version: version, Message: "", Applied: true},
	)
}

func TestIsApplied_WithNotApplied(t *testing.T) {
	db := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(db)
	assert.Nil(t, err)

	version := "20230101010101"
	applied, err := repo.IsApplied(version)
	assert.Nil(t, err)
	assert.Equal(t, applied, false)
}

func TestIsApplied_WithApplied(t *testing.T) {
	db := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(db)
	assert.Nil(t, err)

	version := "20230101010101"
	_, err = db.Session.SQL().InsertInto("bolt_migrations").Columns("version").Values(version).Exec()
	assert.Nil(t, err)

	applied, err := repo.IsApplied(version)
	assert.Nil(t, err)
	assert.Equal(t, applied, true)
}

func TestApply(t *testing.T) {
	testdb := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(testdb)
	assert.Nil(t, err)
	t.Cleanup(func() {
		bolttest.DropTable(t, testdb, "tmp")
	})

	migration := models.NewTimestampMigration(time.Now(), "test")
	err = repo.Apply(`CREATE TABLE tmp(id INT NOT NULL PRIMARY KEY)`, migration)
	assert.Nil(t, err)
	assert.Equal(t, migration.Applied, true)

	exists, err := testdb.Session.Collection("tmp").Exists()
	assert.Nil(t, err)
	assert.True(t, exists)
	applied, err := repo.IsApplied(migration.Version)
	assert.Nil(t, err)
	assert.Equal(t, applied, true)
}

func TestApply_MalformedSql(t *testing.T) {
	db := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(db)
	assert.Nil(t, err)
	migration := models.NewTimestampMigration(time.Now(), "test")

	err = repo.Apply("this is not SQL", migration)

	assert.NotNil(t, err)
	assert.Equal(t, migration.Applied, false)
}

func TestApplyWithTx_ExecErr(t *testing.T) {
	db := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(db)
	assert.Nil(t, err)
	migration := models.NewTimestampMigration(time.Now(), "test")

	err = repo.ApplyWithTx("SELECT 1 FROM abc123donotexist;", migration)

	assert.ErrorContains(t, err, `unable to execute upgrade script`)
}

func TestApplyWithTx_SuccessfullyApplied(t *testing.T) {
	testdb := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(testdb)
	assert.Nil(t, err)
	t.Cleanup(func() {
		bolttest.DropTable(t, testdb, "tmp")
	})

	migration := models.NewTimestampMigration(time.Now(), "test")
	err = repo.ApplyWithTx(`CREATE TABLE tmp(id INT NOT NULL PRIMARY KEY)`, migration)
	assert.Nil(t, err)
	assert.Equal(t, migration.Applied, true)

	exists, err := testdb.Session.Collection("tmp").Exists()
	assert.Nil(t, err)
	assert.True(t, exists)
	applied, err := repo.IsApplied(migration.Version)
	assert.Nil(t, err)
	assert.Equal(t, applied, true)
}

func TestRevert(t *testing.T) {
	testdb := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(testdb)
	assert.Nil(t, err)
	t.Cleanup(func() {
		bolttest.DropTable(t, testdb, "tmp")
	})

	_, err = testdb.Session.SQL().Exec(`CREATE TABLE tmp(id INT NOT NULL PRIMARY KEY)`)
	assert.Nil(t, err)

	migration := models.NewTimestampMigration(time.Now(), "test")
	_, err = testdb.Session.SQL().InsertInto("bolt_migrations").Columns("version").Values(migration.Version).Exec()
	assert.Nil(t, err)
	migration.Applied = true

	err = repo.Revert(`DROP TABLE tmp;`, migration)
	assert.Nil(t, err)
	assert.Equal(t, migration.Applied, false)

	exists, err := testdb.Session.Collection("tmp").Exists()
	assert.ErrorIs(t, err, db.ErrCollectionDoesNotExist)
	assert.False(t, exists)
	count, err := testdb.Session.Collection("bolt_migrations").Count()
	assert.Nil(t, err)
	assert.Equal(t, count, uint64(0))
}

func TestRevert_MalformedSql(t *testing.T) {
	db := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(db)
	assert.Nil(t, err)
	migration := models.NewTimestampMigration(time.Now(), "test")
	migration.Applied = true

	err = repo.Revert("this is not SQL", migration)
	assert.ErrorContains(t, err, "unable to execute downgrade script")
	assert.Equal(t, migration.Applied, true)
}

func TestRevertWithTx_ExecErr(t *testing.T) {
	db := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(db)
	assert.Nil(t, err)
	migration := models.NewTimestampMigration(time.Now(), "test")

	err = repo.RevertWithTx("DROP TABLE abc123donotexist;", migration)

	assert.ErrorContains(t, err, `unable to execute downgrade script`)
}

func TestRevertWithTx_SuccessfullyReverted(t *testing.T) {
	testdb := bolttest.NewTestDB(t)
	repo, err := repositories.NewMigrationDBRepo(testdb)
	assert.Nil(t, err)
	t.Cleanup(func() {
		bolttest.DropTable(t, testdb, "tmp")
	})

	_, err = testdb.Session.SQL().Exec(`CREATE TABLE tmp(id INT NOT NULL PRIMARY KEY)`)
	assert.Nil(t, err)

	migration := models.NewTimestampMigration(time.Now(), "test")
	_, err = testdb.Session.SQL().InsertInto("bolt_migrations").Columns("version").Values(migration.Version).Exec()
	assert.Nil(t, err)
	migration.Applied = true

	err = repo.RevertWithTx(`DROP TABLE tmp;`, migration)
	assert.Nil(t, err)
	assert.Equal(t, migration.Applied, false)

	exists, err := testdb.Session.Collection("tmp").Exists()
	assert.ErrorIs(t, err, db.ErrCollectionDoesNotExist)
	assert.False(t, exists)
	count, err := testdb.Session.Collection("bolt_migrations").Count()
	assert.Nil(t, err)
	assert.Equal(t, count, uint64(0))
}
