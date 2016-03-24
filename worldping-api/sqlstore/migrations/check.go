package migrations

import (
	"fmt"

	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addCheckMigrations(mg *migrator.Migrator) {
	checkV1 := migrator.Table{
		Name: "check",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "endpoint_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "task_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "owner", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "type", Type: migrator.DB_NVarchar, Length: 64},
			{Name: "frequency", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "enabled", Type: migrator.DB_Bool, Nullable: false},
			{Name: "settings", Type: migrator.DB_NVarchar, Length: 2048, Nullable: false},
			{Name: "state", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "state_check", Type: migrator.DB_DateTime, Nullable: true},
			{Name: "state_change", Type: migrator.DB_DateTime, Nullable: false},
			{Name: "health_settings", Type: migrator.DB_NVarchar, Length: 2048, Nullable: true, Default: ""},
			{Name: "created", Type: migrator.DB_DateTime, Nullable: false},
			{Name: "updated", Type: migrator.DB_DateTime, Nullable: false},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"endpoint_id", "owner"}},
		},
	}
	mg.AddMigration("create check table v1", migrator.NewAddTableMigration(checkV1))
	for _, index := range checkV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(checkV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(checkV1, index))
	}
}
