package db

import (
	"sukumad/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	log "github.com/sirupsen/logrus"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //import postgres
)

var db *sqlx.DB

func init() {
	psqlInfo := config.AppConfig.DBURL
	//
	var err error
	db, err = ConnectDB(psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
}

// ConnectDB ...
func ConnectDB(dataSourceName string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dataSourceName)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	return db, nil
}

// GetDB ...
func GetDB() *sqlx.DB {
	return db
}

func RunMigrations() error {
	migrationsDir := config.AppConfig.MigrationsDirectory
	log.Infof("Running migrations from %s", migrationsDir)
	m, err := migrate.New(
		migrationsDir,
		config.AppConfig.DBURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
