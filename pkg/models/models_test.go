package models

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/storage"
	storagedb "github.com/yubo/apiserver/pkg/storage/db"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/orm"

	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

var (
	dsn       string
	driver    string
	available bool
)

func envDef(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func runTests(t *testing.T, tests ...func()) {
	// See https://github.com/go-sql-driver/mysql/wiki/Testing
	driver := envDef("TEST_DB_DRIVER", "sqlite3")
	dsn := envDef("TEST_DB_DSN", "file:test.db?cache=shared&mode=memory")

	db, err := orm.Open(driver, dsn)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	SetStorage(storagedb.New(db), "test_")

	for _, test := range tests {
		test()
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

	runTests(t, func() {
		roles := &role{store: NewStore("role")}
		roles.store.Drop()
		defer roles.store.Drop()

		AutoMigrate("role", roles.NewObj())

		t.Run("create role", func(t *testing.T) {
			ret, err := roles.Create(context.TODO(), testRole)
			assert.NoError(t, err)
			assert.NotNil(t, ret)
		})

		t.Run("get role", func(t *testing.T) {
			ret, err := roles.Get(context.TODO(), "test-role")
			assert.NotNil(t, ret)
			assert.NoError(t, err)
		})

		t.Run("list roles", func(t *testing.T) {
			list, err := roles.List(context.TODO(), storage.ListOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, list)
		})

		testRole.Rules[0].Verbs = []string{"get"}
		t.Run("update role", func(t *testing.T) {
			ret, err := roles.Update(context.TODO(), testRole)
			assert.NotNil(t, ret)
			assert.NoError(t, err)
		})

		t.Run("delete role", func(t *testing.T) {
			ret, err := roles.Delete(context.TODO(), "test-role")
			assert.NotNil(t, ret)
			assert.NoError(t, err)
		})
	})
}
