package migrations

import (
	"fmt"

	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addEndpointTagMigrations(mg *migrator.Migrator) {
	endpointTagV1 := migrator.Table{
		Name: "endpoint_tag",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "endpoint_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "owner", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "tag", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "created", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"owner", "endpoint_id"}},
			{Cols: []string{"owner", "tag"}},
		},
	}
	mg.AddMigration("create endpoint_tag table v1", migrator.NewAddTableMigration(endpointTagV1))
	for _, index := range endpointTagV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(endpointTagV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(endpointTagV1, index))
	}
}
