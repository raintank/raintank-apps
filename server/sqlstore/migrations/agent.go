package migrations

import (
	"fmt"

	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addAgentMigrations(mg *migrator.Migrator) {
	agentV1 := migrator.Table{
		Name: "agent",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "name", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "slug", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "password", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "owner", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "enabled", Type: migrator.DB_Bool},
			{Name: "public", Type: migrator.DB_Bool},
			{Name: "created", Type: migrator.DB_DateTime},
			{Name: "updated", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"owner", "public"}},
		},
	}
	mg.AddMigration("create agent table v1", migrator.NewAddTableMigration(agentV1))
	for _, index := range agentV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(agentV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(agentV1, index))
	}
}
