package migrations

import (
	"fmt"

	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addEndpointMigrations(mg *migrator.Migrator) {
	endpointV1 := migrator.Table{
		Name: "endpoint",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "owner", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "name", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "slug", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "created", Type: migrator.DB_DateTime},
			{Name: "updated", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"owner", "id"}},
			{Cols: []string{"slug", "owner"}, Type: migrator.UniqueIndex},
			{Cols: []string{"name", "owner"}, Type: migrator.UniqueIndex},
		},
	}
	mg.AddMigration("create endpoint table v1", migrator.NewAddTableMigration(endpointV1))
	for _, index := range endpointV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(endpointV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(endpointV1, index))
	}
}
