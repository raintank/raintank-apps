package sqlstore

import (
	"fmt"
	"os"
	"path"

	"github.com/go-xorm/xorm"
	"github.com/grafana/grafana/pkg/log"
	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
	_ "github.com/mattn/go-sqlite3"
	"github.com/raintank/raintank-apps/task-server/sqlstore/migrations"
)

var (
	x       *xorm.Engine
	dialect migrator.Dialect
)

type session struct {
	*xorm.Session
	transaction bool
	complete    bool
}

func newSession(transaction bool, table string) (*session, error) {
	if !transaction {
		return &session{Session: x.Table(table)}, nil
	}
	sess := session{Session: x.NewSession(), transaction: true}
	if err := sess.Begin(); err != nil {
		return nil, err
	}
	sess.Table(table)
	return &sess, nil
}

func (sess *session) Complete() {
	if sess.transaction {
		if err := sess.Commit(); err == nil {
			sess.complete = true
		}
	}
}

func (sess *session) Cleanup() {
	if sess.transaction {
		if !sess.complete {
			sess.Rollback()
		}
		sess.Close()
	}
}

func NewEngine(dbPath string) {
	x, err := getEngine(dbPath)

	if err != nil {
		log.Fatal(3, "Sqlstore: Fail to connect to database: %v", err)
	}
	err = SetEngine(x, true)
	if err != nil {
		log.Fatal(3, "fail to initialize orm engine: %v", err)
	}
}

func SetEngine(engine *xorm.Engine, enableLog bool) (err error) {
	x = engine
	dialect = migrator.NewDialect(x.DriverName())

	migrator := migrator.NewMigrator(x)
	migrator.LogLevel = 2
	migrations.AddMigrations(migrator)

	if err := migrator.Start(); err != nil {
		return fmt.Errorf("Sqlstore::Migration failed err: %v\n", err)
	}

	logPath := path.Join("/tmp", "xorm.log")
	os.MkdirAll(path.Dir(logPath), os.ModePerm)
	f, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("sqlstore.init(fail to create xorm.log): %v", err)
	}
	x.SetLogger(xorm.NewSimpleLogger(f))
	x.ShowSQL(true)

	return nil
}

func getEngine(dbPath string) (*xorm.Engine, error) {
	os.MkdirAll(path.Dir(dbPath), os.ModePerm)
	cnnstr := "file:" + dbPath + "?cache=shared&mode=rwc&_loc=Local"

	return xorm.NewEngine("sqlite3", cnnstr)
}
