package migrations

import (
	"fmt"

	"github.com/raintank/worldping-api/pkg/services/sqlstore/migrator"
)

func addAgentMigrations(mg *migrator.Migrator) {
	agentV1 := migrator.Table{
		Name: "agent",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "name", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "org_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "enabled", Type: migrator.DB_Bool},
			{Name: "enabled_change", Type: migrator.DB_DateTime},
			{Name: "online", Type: migrator.DB_Bool},
			{Name: "online_change", Type: migrator.DB_DateTime},
			{Name: "public", Type: migrator.DB_Bool},
			{Name: "created", Type: migrator.DB_DateTime},
			{Name: "updated", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"org_id", "public"}},
			{Cols: []string{"name", "org_id"}, Type: migrator.UniqueIndex},
		},
	}
	mg.AddMigration("create agent table v1", migrator.NewAddTableMigration(agentV1))
	for _, index := range agentV1.Indices {
		migrationID := fmt.Sprintf("create index %s - %s", index.XName(agentV1.Name), "v1")
		mg.AddMigration(migrationID, migrator.NewAddIndexMigration(agentV1, index))
	}
}
