package migrations

import (
	"fmt"

	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addAgentTagMigrations(mg *migrator.Migrator) {
	agentTagV1 := migrator.Table{
		Name: "agent_tag",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "agent_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "owner", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "tag", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "created", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"owner", "agent_id"}},
			{Cols: []string{"owner", "tag"}},
		},
	}
	mg.AddMigration("create agent_tag table v1", migrator.NewAddTableMigration(agentTagV1))
	for _, index := range agentTagV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(agentTagV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(agentTagV1, index))
	}
}
