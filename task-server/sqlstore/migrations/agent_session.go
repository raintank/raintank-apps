package migrations

import (
	"fmt"

	"github.com/raintank/worldping-api/pkg/services/sqlstore/migrator"
)

func addAgentSessionMigrations(mg *migrator.Migrator) {
	agentSessionV1 := migrator.Table{
		Name: "agent_session",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_NVarchar, Length: 64, IsPrimaryKey: true},
			{Name: "agent_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "version", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "server", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "remote_ip", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "created", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"agent_id"}},
			{Cols: []string{"server"}},
		},
	}
	mg.AddMigration("create agent_session table v1", migrator.NewAddTableMigration(agentSessionV1))
	for _, index := range agentSessionV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(agentSessionV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(agentSessionV1, index))
	}
}
