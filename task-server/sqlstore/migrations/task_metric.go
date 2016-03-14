package migrations

import (
	"fmt"

	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addTaskMetricMigrations(mg *migrator.Migrator) {
	taskMetricV1 := migrator.Table{
		Name: "task_metric",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "task_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "namespace", Type: migrator.DB_NVarchar, Length: 255, Nullable: false},
			{Name: "version", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "created", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"task_id", "namespace", "version"}, Type: migrator.UniqueIndex},
		},
	}
	mg.AddMigration("create task_metric table v1", migrator.NewAddTableMigration(taskMetricV1))
	for _, index := range taskMetricV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(taskMetricV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(taskMetricV1, index))
	}
}
