package orm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yubo/apiserver/pkg/sessions"
	"github.com/yubo/apiserver/pkg/sessions/tester"
	"github.com/yubo/golib/orm"

	_ "github.com/yubo/golib/orm/sqlite"
)

var newStore = func(t *testing.T, o *sessions.Options) sessions.Store {
	if o == nil {
		o = &sessions.Options{
			Path:   "/",
			MaxAge: 30 * 24 * 3600,
		}
	}
	o.KeyPairs = [][]byte{[]byte("secret")}

	db, err := orm.Open("sqlite3", "file:test.db?cache=shared&mode=memory&parseTime=true")
	require.NoError(t, err)

	cf := newConfig()
	cf.Options = o
	cf.Orm = db

	store, err := NewStore(cf)
	require.NoError(t, err)

	return store
}

func TestGorm_SessionGetSet(t *testing.T) {
	tester.GetSet(t, newStore)
}

func TestGorm_SessionDeleteKey(t *testing.T) {
	tester.DeleteKey(t, newStore)
}

func TestGorm_SessionFlashes(t *testing.T) {
	tester.Flashes(t, newStore)
}

func TestGorm_SessionClear(t *testing.T) {
	tester.Clear(t, newStore)
}

func TestGorm_SessionOptions(t *testing.T) {
	tester.Options(t, newStore)
}

func TestGorm_SessionMany(t *testing.T) {
	tester.Many(t, newStore)
}
