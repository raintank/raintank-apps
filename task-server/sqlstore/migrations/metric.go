package migrations

import (
	"fmt"

	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addMetricMigrations(mg *migrator.Migrator) {
	metricV1 := migrator.Table{
		Name: "metric",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, Length: 255, IsPrimaryKey: true},
			{Name: "namespace", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "owner", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "public", Type: migrator.DB_Bool},
			{Name: "policy", Type: migrator.DB_Text},
			{Name: "version", Type: migrator.DB_BigInt},
			{Name: "created", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"owner", "public", "namespace", "version"}, Type: migrator.UniqueIndex},
		},
	}
	mg.AddMigration("create metric table v1", migrator.NewAddTableMigration(metricV1))
	for _, index := range metricV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(metricV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(metricV1, index))
	}
}
