package migrations

import . "github.com/raintank/worldping-api/pkg/services/sqlstore/migrator"

// --- Migration Guide line ---
// 1. Never change a migration that is committed and pushed to master
// 2. Always add new migrations (to change or undo previous migrations)
// 3. Some migraitons are not yet written (rename column, table, drop table, index etc)

func AddMigrations(mg *Migrator) {
	addMigrationLogMigrations(mg)

	addAgentMigrations(mg)
	addAgentTagMigrations(mg)
	addAgentSessionMigrations(mg)
	addTaskMigrations(mg)

	addRouteByIdIndexMigrations(mg)
	addRouteByTagIndexMigrations(mg)
	addRouteByAnyIndexMigrations(mg)

}
