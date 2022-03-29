package session

import (
	"bytes"
	"context"
	"fmt"
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
	storagedb "github.com/yubo/apiserver/pkg/storage/db"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/util/clock"

	_ "github.com/yubo/golib/orm/sqlite"
)

var (
	driver   = "sqlite3"
	dsn      = "file:test.db?cache=shared&mode=memory"
	db       orm.DB
	sessions *session
)

func TestMain(m *testing.M) {
	if err := testInit(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	code := m.Run()

	db.Close()

	os.Exit(code)
}

func testInit() error {
	var err error

	// init storage
	// init models
	db, err = orm.Open(driver, dsn)
	if err != nil {
		return fmt.Errorf("open %s err %s", dsn, err)
	}

	models.SetStorage(storagedb.New(db), "test_")
	sessions = &session{store: models.NewStore("session")}
	sessions.store.Drop()

	return nil
}

func TestDbSession(t *testing.T) {
	var (
		sm      types.SessionManager
		sessCtx types.SessionContext
		err     error
		sid     string
	)

	cf := NewConfig()

	if sm, err = NewSessionManager(cf, WithModel(sessions)); err != nil {
		t.Fatalf("error NewSession: %s", err.Error())
	}
	defer sm.Stop()

	models.AutoMigrate("session", sessions.NewObj())
	defer sessions.store.Drop()

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
}

func TestDbSessionGC(t *testing.T) {
	cf := NewConfig()
	clock := &clock.FakeClock{}
	clock.SetTime(time.Now())
	orm.SetClock(clock)

	sm, err := NewSessionManager(cf, WithModel(sessions), WithClock(clock))
	assert.NoError(t, err)
	sm.GC()
	defer sm.Stop()

	models.AutoMigrate("session", sessions.NewObj())
	defer sessions.store.Drop()

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
}
