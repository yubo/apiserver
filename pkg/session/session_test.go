package session

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/session/types"
	"github.com/yubo/apiserver/pkg/storage"
	dbstore "github.com/yubo/apiserver/pkg/storage/db"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/util/clock"

	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

func envDef(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func runTests(t *testing.T, tests ...func(*session)) {
	driver := envDef("TEST_DB_DRIVER", "sqlite3")
	dsn := envDef("TEST_DB_DSN", "file:test.db?cache=shared&mode=memory")

	db, err := orm.Open(driver, dsn)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	store := dbstore.New(db)
	defer store.Drop("session")

	m := models.NewModels(store)
	m.Register(&session{})

	sessions := &session{store: m.NewModelStore("session")}
	store.AutoMigrate("session", sessions.NewObj())

	for _, test := range tests {
		test(sessions)
	}
}

func TestDbSession(t *testing.T) {
	var (
		sm      types.SessionManager
		sessCtx types.SessionContext
		err     error
		sid     string
	)

	runTests(t, func(sessions *session) {
		cf := NewConfig()

		if sm, err = NewSessionManager(cf, WithModel(sessions)); err != nil {
			t.Fatalf("error NewSession: %s", err.Error())
		}
		defer sm.Stop()

		r, _ := http.NewRequest("GET", "", bytes.NewBuffer([]byte{}))
		w := httptest.NewRecorder()

		sessCtx, err = sm.Start(w, r)
		assert.NoError(t, err)

		list, err := sessions.List(context.TODO(), storage.ListOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(list))

		sessCtx.Set("abc", "11223344")
		err = sessCtx.Update(w)
		assert.NoError(t, err)

		sid = sessCtx.Sid()

		// new request
		r, _ = http.NewRequest("GET", "", bytes.NewBuffer([]byte{}))
		w = httptest.NewRecorder()

		cookie := &http.Cookie{
			Name:     cf.CookieName,
			Value:    url.QueryEscape(sid),
			Path:     "/",
			HttpOnly: cf.HttpOnly,
			Domain:   cf.Domain,
		}
		if cf.CookieLifetime.Duration > 0 {
			cookie.Expires = time.Now().Add(cf.CookieLifetime.Duration)
		}
		http.SetCookie(w, cookie)
		r.AddCookie(cookie)

		sessCtx, err = sm.Start(w, r)
		assert.NoError(t, err)

		assert.Equal(t, "11223344", sessCtx.Get("abc"))

		sessCtx.Set("abc", "22334455")
		assert.Equal(t, "22334455", sessCtx.Get("abc"))

		list, err = sessions.List(context.TODO(), storage.ListOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(list))

		sm.Destroy(w, r)
		list, err = sessions.List(context.TODO(), storage.ListOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(list))
	})
}

func TestDbSessionGC(t *testing.T) {
	runTests(t, func(sessions *session) {
		cf := NewConfig()
		clock := &clock.FakeClock{}
		clock.SetTime(time.Now())
		orm.SetClock(clock)

		sm, err := NewSessionManager(cf, WithModel(sessions), WithClock(clock))

		assert.NoError(t, err)
		sm.GC()
		defer sm.Stop()

		r, _ := http.NewRequest("GET", "", bytes.NewBuffer([]byte{}))
		w := httptest.NewRecorder()

		_, err = sm.Start(w, r)
		assert.NoError(t, err)

		list, err := sessions.List(context.TODO(), storage.ListOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(list))

		clock.SetTime(clock.Now().Add(time.Hour * 25))
		time.Sleep(100 * time.Millisecond)

		list, err = sessions.List(context.TODO(), storage.ListOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(list))
	})
}
