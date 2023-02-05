package models

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yubo/apiserver/pkg/apis/rbac"
	dbstore "github.com/yubo/apiserver/pkg/storage/db"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/orm"

	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

func envDef(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func runTests(t *testing.T, tests ...func(*Role)) {
	// See https://github.com/go-sql-driver/mysql/wiki/Testing
	driver := envDef("TEST_DB_DRIVER", "sqlite3")
	dsn := envDef("TEST_DB_DSN", "file:test.db?cache=shared&mode=memory")

	db, err := orm.Open(driver, dsn)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	store := dbstore.New(db)
	defer store.Drop(context.Background(), "role")

	m := NewModels(store)
	m.Register(&Role{})

	roles := &Role{DB: db}
	store.AutoMigrate(context.Background(), "role", roles.NewObj())

	for _, test := range tests {
		test(roles)
	}
}

func TestRole(t *testing.T) {
	testRole := &rbac.Role{
		ObjectMeta: api.ObjectMeta{
			Name: "test-role",
		},
		Rules: []rbac.PolicyRule{{
			Verbs:     []string{"get", "list"},
			Resources: []string{"users", "status"},
		}},
	}

	//orm.DEBUG = true
	runTests(t, func(roles *Role) {
		t.Run("create role", func(t *testing.T) {
			err := roles.Create(context.TODO(), testRole)
			assert.NoError(t, err)
		})

		t.Run("get role", func(t *testing.T) {
			ret, err := roles.Get(context.TODO(), "test-role")
			assert.NoError(t, err)
			assert.NotNil(t, ret)
		})

		t.Run("list roles", func(t *testing.T) {
			list, err := roles.List(context.TODO(), api.GetListOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, list)
		})

		testRole.Rules[0].Verbs = []string{"get"}
		t.Run("update role", func(t *testing.T) {
			err := roles.Update(context.TODO(), testRole)
			assert.NoError(t, err)
		})

		t.Run("delete role", func(t *testing.T) {
			err := roles.Delete(context.TODO(), "test-role")
			assert.NoError(t, err)

		})
	})
}
