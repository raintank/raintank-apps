package migrations

import (
	"fmt"

	"github.com/raintank/worldping-api/pkg/services/sqlstore/migrator"
)

func addRouteByIdIndexMigrations(mg *migrator.Migrator) {
	routeIndexV1 := migrator.Table{
		Name: "route_by_id_index",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "task_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "agent_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "created", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"task_id", "agent_id"}, Type: migrator.UniqueIndex},
		},
	}
	mg.AddMigration("create route_by_id_index table v1", migrator.NewAddTableMigration(routeIndexV1))
	for _, index := range routeIndexV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(routeIndexV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(routeIndexV1, index))
	}
}

func addRouteByTagIndexMigrations(mg *migrator.Migrator) {
	routeIndexV1 := migrator.Table{
		Name: "route_by_tag_index",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "org_id", Type: migrator.DB_BigInt, Nullable: true},
			{Name: "task_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "tag", Type: migrator.DB_NVarchar, Length: 255, Nullable: false},
			{Name: "created", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"task_id", "tag"}},
		},
	}
	mg.AddMigration("create route_by_id_index table v1", migrator.NewAddTableMigration(routeIndexV1))
	for _, index := range routeIndexV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(routeIndexV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(routeIndexV1, index))
	}
}

func addRouteByAnyIndexMigrations(mg *migrator.Migrator) {
	routeIndexV1 := migrator.Table{
		Name: "route_by_any_index",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "task_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "agent_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "created", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"task_id", "agent_id"}, Type: migrator.UniqueIndex},
		},
	}
	mg.AddMigration("create route_by_any_index table v1", migrator.NewAddTableMigration(routeIndexV1))
	for _, index := range routeIndexV1.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(routeIndexV1.Name), "v1")
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(routeIndexV1, index))
	}
}
