package migrations

import (
	"fmt"

	"github.com/raintank/worldping-api/pkg/services/sqlstore/migrator"
)

func addTaskMigrations(mg *migrator.Migrator) {
	taskV1 := migrator.Table{
		Name: "task",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "name", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "config", Type: migrator.DB_Text},
			{Name: "interval", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "org_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "enabled", Type: migrator.DB_Bool},
			{Name: "route", Type: migrator.DB_Text, Nullable: false},
			{Name: "created", Type: migrator.DB_DateTime},
			{Name: "updated", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"org_id", "name"}, Type: migrator.UniqueIndex},
		},
	}
	mg.AddMigration("create task table v1", migrator.NewAddTableMigration(taskV1))
	for _, index := range taskV1.Indices {
		migrationID := fmt.Sprintf("create index %s - %s", index.XName(taskV1.Name), "v1")
		mg.AddMigration(migrationID, migrator.NewAddIndexMigration(taskV1, index))
	}
	// add task type
	migration := migrator.NewAddColumnMigration(taskV1, &migrator.Column{
		Name: "task_type", Type: migrator.DB_NVarchar, Length: 255, Nullable: true,
	})
	//lamigra.OnSuccess = bubba(nil)
	mg.AddMigration("task add taskType field", migration)
}
