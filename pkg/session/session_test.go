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
	"github.com/yubo/apiserver/pkg/session/types"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/util/clock"
	testingclock "github.com/yubo/golib/util/clock/testing"

	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

func envDef(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func runTests(t *testing.T, tests ...func(*SessionConn)) {
	driver := envDef("TEST_DB_DRIVER", "sqlite3")
	dsn := envDef("TEST_DB_DSN", "file:test.db?cache=shared&mode=memory")

	db, err := orm.Open(driver, dsn)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	sessionConn := &SessionConn{DB: db}
	db.AutoMigrate(context.Background(), sessionConn.NewObj(), orm.WithTable(sessionConn.Name()))

	opts, _ := orm.NewOptions(orm.WithTable(sessionConn.Name()))
	defer db.DropTable(context.Background(), opts)

	for _, test := range tests {
		test(sessionConn)
	}
}

func TestDbSession(t *testing.T) {
	var (
		sessCtx types.Session
		err     error
		sid     string
	)

	runTests(t, func(sessions *SessionConn) {
		cf := newConfig()
		ctx, cancel := context.WithCancel(context.TODO())

		sm := &manager{
			ctx:     ctx,
			config:  cf,
			session: sessions,
			clock:   clock.RealClock{},
		}
		defer cancel()

		req, _ := http.NewRequest("GET", "", bytes.NewBuffer([]byte{}))
		w := httptest.NewRecorder()

		sessCtx, err = sm.Start(w, req)
		assert.NoError(t, err)

		list, err := sessions.List(context.TODO(), &api.GetListOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(list))

		sessCtx.Set("abc", "11223344")
		err = sessCtx.Update(w)
		assert.NoError(t, err)

		sid = sessCtx.Sid()

		// new request
		req, _ = http.NewRequest("GET", "", bytes.NewBuffer([]byte{}))
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
		req.AddCookie(cookie)

		sessCtx, err = sm.Start(w, req)
		assert.NoError(t, err)

		assert.Equal(t, "11223344", sessCtx.Get("abc"))

		sessCtx.Set("abc", "22334455")
		assert.Equal(t, "22334455", sessCtx.Get("abc"))

		list, err = sessions.List(context.TODO(), &api.GetListOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(list))

		err = sm.Destroy(w, req)
		assert.NoError(t, err)

		list, err = sessions.List(context.TODO(), &api.GetListOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(list))
	})
}

func TestDbSessionGC(t *testing.T) {
	runTests(t, func(sessions *SessionConn) {
		ctx, cancel := context.WithCancel(context.TODO())
		cf := newConfig()
		clock := &testingclock.FakeClock{}
		clock.SetTime(time.Now())
		orm.SetClock(clock)

		sm := &manager{
			ctx:     ctx,
			config:  cf,
			session: sessions,
			clock:   clock,
		}
		defer cancel()

		sm.GC()

		r, _ := http.NewRequest("GET", "", bytes.NewBuffer([]byte{}))
		w := httptest.NewRecorder()

		_, err := sm.Start(w, r)
		assert.NoError(t, err)

		list, err := sessions.List(context.TODO(), &api.GetListOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(list))

		clock.SetTime(clock.Now().Add(time.Hour * 25))
		time.Sleep(100 * time.Millisecond)

		list, err = sessions.List(context.TODO(), &api.GetListOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(list))
	})
}
