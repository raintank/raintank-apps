package migrations

import (
	"fmt"

	"github.com/go-xorm/xorm"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/worldping-api/pkg/log"
	"github.com/raintank/worldping-api/pkg/services/sqlstore/migrator"
)

func addTaskMigrations(mg *migrator.Migrator) {
	taskV1 := migrator.Table{
		Name: "task",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "name", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "config", Type: migrator.DB_Text},
			{Name: "interval", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "org_id", Type: migrator.DB_BigInt, Nullable: false},
			{Name: "enabled", Type: migrator.DB_Bool},
			{Name: "route", Type: migrator.DB_Text, Nullable: false},
			{Name: "created", Type: migrator.DB_DateTime},
			{Name: "updated", Type: migrator.DB_DateTime},
		},
		Indices: []*migrator.Index{
			{Cols: []string{"org_id", "name"}, Type: migrator.UniqueIndex},
		},
	}
	mg.AddMigration("create task table v1", migrator.NewAddTableMigration(taskV1))
	for _, index := range taskV1.Indices {
		migrationID := fmt.Sprintf("create index %s - %s", index.XName(taskV1.Name), "v1")
		mg.AddMigration(migrationID, migrator.NewAddIndexMigration(taskV1, index))
	}
	// add task type
	migration := migrator.NewAddColumnMigration(taskV1, &migrator.Column{
		Name: "task_type", Type: migrator.DB_NVarchar, Length: 255, Nullable: true,
	})
	migration.OnSuccess = func(sess *xorm.Session) error {
		log.Info("setting TaskType on all tasks in the DB.")
		// iterate over every task, and copy the config map key to be the taskType.
		sess.Table("task")
		var t []*model.Task
		err := sess.Find(&t)
		if err != nil {
			log.Error(3, "failed to get list of tasks. %s", err)
			return err
		}
		log.Info("found %d tasks in the DB.", len(t))
		for _, task := range t {
			keys := make([]string, 0)
			for k := range task.Config {
				keys = append(keys, k)
			}
			if len(keys) > 0 {
				task.TaskType = keys[0]
			}
			_, err := sess.Id(task.Id).Update(task)
			if err != nil {
				log.Error(3, "failed to update task with new taskType. %s", err)
				return err
			}
		}

		return err
	}
	mg.AddMigration("task add taskType field", migration)
}
