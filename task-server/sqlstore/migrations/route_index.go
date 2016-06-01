package migrations

import (
	"fmt"

	"github.com/go-xorm/xorm"
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

	// add health settings
	migration := migrator.NewAddColumnMigration(routeIndexV1, &migrator.Column{
		Name: "org_id", Type: migrator.DB_BigInt, Nullable: true,
	})
	migration.OnSuccess = func(sess *xorm.Session) error {
		rawSQL := "REPLACE INTO route_by_tag_index SELECT tt.id, tt.task_id, tt.tag, tt.created, t.org_id from route_by_tag_index as tt JOIN task as t on tt.task_id=t.id"
		sess.Table("route_by_tag_index")
		_, err := sess.Exec(rawSQL)
		return err
	}
	mg.AddMigration("route_by_tag_index add org-id v1", migration)
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
