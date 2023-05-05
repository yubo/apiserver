// Package tester is a package to test each packages of session stores, such as
// cookie, redis, memcached, mongo, memstore.  You can use this to test your own session
// stores.
package tester

import (
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yubo/apiserver/pkg/sessions"
)

type storeFactory func(*testing.T, *sessions.Options) sessions.Store

const sessionName = "mysession"

const ok = "ok"

type userInfo struct {
	Name  string
	Group []string
	Extra map[string]string
}

var (
	testUser = &userInfo{"tom", []string{"dev", "admin", "sre"}, map[string]string{"age": "28"}}
)

func init() {
	gob.Register(new(userInfo))
}

func GetSet(t *testing.T, newStore storeFactory) {
	mux := http.NewServeMux()
	mux.HandleFunc("/set", func(w http.ResponseWriter, req *http.Request) {
		session := sessions.Default(req.Context())
		session.Set("key", ok)
		session.Set("testUser", testUser)
		err := session.Save()
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/get", func(w http.ResponseWriter, req *http.Request) {
		session := sessions.Default(req.Context())
		require.Equal(t, ok, session.Get("key"), "Session writing failed")
		require.Equal(t, testUser, session.Get("testUser"), "Session writing failed")

		err := session.Save()
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	handler := sessions.Sessions(mux, sessionName, newStore(t, nil))

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set", nil)
	handler.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/get", nil)
	copyCookies(req2, res1)
	handler.ServeHTTP(res2, req2)
}

func DeleteKey(t *testing.T, newStore storeFactory) {
	mux := http.NewServeMux()
	mux.HandleFunc("/set", func(w http.ResponseWriter, req *http.Request) {
		session := sessions.Default(req.Context())
		session.Set("key", ok)
		_ = session.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/delete", func(w http.ResponseWriter, req *http.Request) {
		session := sessions.Default(req.Context())
		session.Delete("key")
		_ = session.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/get", func(w http.ResponseWriter, req *http.Request) {
		session := sessions.Default(req.Context())
		if session.Get("key") != nil {
			t.Error("Session deleting failed")
		}

		_ = session.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	handler := sessions.Sessions(mux, sessionName, newStore(t, nil))

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set", nil)
	handler.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/delete", nil)
	copyCookies(req2, res1)
	handler.ServeHTTP(res2, req2)

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/get", nil)
	copyCookies(req3, res2)
	handler.ServeHTTP(res3, req3)
}

func Flashes(t *testing.T, newStore storeFactory) {
	mux := http.NewServeMux()
	mux.HandleFunc("/set", func(w http.ResponseWriter, req *http.Request) {
		session := sessions.Default(req.Context())
		session.AddFlash(ok)
		_ = session.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/flash", func(w http.ResponseWriter, req *http.Request) {
		session := sessions.Default(req.Context())
		l := len(session.Flashes())
		if l != 1 {
			t.Error("Flashes count does not equal 1. Equals ", l)
		}
		_ = session.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/check", func(w http.ResponseWriter, req *http.Request) {
		session := sessions.Default(req.Context())
		l := len(session.Flashes())
		if l != 0 {
			t.Error("flashes count is not 0 after reading. Equals ", l)
		}
		_ = session.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	handler := sessions.Sessions(mux, sessionName, newStore(t, nil))

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set", nil)
	handler.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/flash", nil)
	copyCookies(req2, res1)
	handler.ServeHTTP(res2, req2)

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/check", nil)
	copyCookies(req3, res2)
	handler.ServeHTTP(res3, req3)
}

func Clear(t *testing.T, newStore storeFactory) {
	data := map[string]string{
		"key": "val",
		"foo": "bar",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/set", func(w http.ResponseWriter, req *http.Request) {
		session := sessions.Default(req.Context())
		for k, v := range data {
			session.Set(k, v)
		}
		session.Clear()
		_ = session.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/check", func(w http.ResponseWriter, req *http.Request) {
		session := sessions.Default(req.Context())
		for k, v := range data {
			if session.Get(k) == v {
				t.Fatal("Session clear failed")
			}
		}
		_ = session.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	handler := sessions.Sessions(mux, sessionName, newStore(t, nil))

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set", nil)
	handler.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/check", nil)
	copyCookies(req2, res1)
	handler.ServeHTTP(res2, req2)
}

func Options(t *testing.T, newStore storeFactory) {
	mux := http.NewServeMux()
	mux.HandleFunc("/domain", func(w http.ResponseWriter, req *http.Request) {
		sess := sessions.Default(req.Context())
		sess.Set("key", ok)
		sess.Options(sessions.Options{
			Path: "/foo/bar/bat",
		})
		_ = sess.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/path", func(w http.ResponseWriter, req *http.Request) {
		sess := sessions.Default(req.Context())
		sess.Set("key", ok)
		_ = sess.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/set", func(w http.ResponseWriter, req *http.Request) {
		sess := sessions.Default(req.Context())
		sess.Set("key", ok)
		_ = sess.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/expire", func(w http.ResponseWriter, req *http.Request) {
		sess := sessions.Default(req.Context())
		sess.Options(sessions.Options{
			MaxAge: -1,
		})
		_ = sess.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/check", func(w http.ResponseWriter, req *http.Request) {
		sess := sessions.Default(req.Context())
		val := sess.Get("key")
		if val != nil {
			t.Fatal("Session expiration failed")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})
	mux.HandleFunc("/sameSite", func(w http.ResponseWriter, req *http.Request) {
		sess := sessions.Default(req.Context())
		sess.Set("key", ok)
		sess.Options(sessions.Options{
			SameSite: http.SameSiteStrictMode,
		})
		_ = sess.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))

	})

	store := newStore(t, &sessions.Options{Domain: "localhost"})

	handler := sessions.Sessions(mux, sessionName, store)

	res0 := httptest.NewRecorder()
	req0, _ := http.NewRequest("GET", "/sameSite", nil)
	handler.ServeHTTP(res0, req0)
	s := strings.Split(res0.Header().Get("Set-Cookie"), ";")
	if s[1] != " SameSite=Strict" {
		t.Error("Error writing samesite with options:", s[1])
	}

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/domain", nil)
	handler.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/path", nil)
	handler.ServeHTTP(res2, req2)

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/set", nil)
	handler.ServeHTTP(res3, req3)

	res4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/expire", nil)
	handler.ServeHTTP(res4, req4)

	res5 := httptest.NewRecorder()
	req5, _ := http.NewRequest("GET", "/check", nil)
	handler.ServeHTTP(res5, req5)

	for _, c := range res1.Header().Values("Set-Cookie") {
		s := strings.Split(c, ";")
		if s[1] != " Path=/foo/bar/bat" {
			t.Error("Error writing path with options:", s[1])
		}
	}

	for _, c := range res2.Header().Values("Set-Cookie") {
		s := strings.Split(c, ";")
		if s[1] != " Domain=localhost" {
			t.Error("Error writing domain with options:", s[1])
		}
	}
}

func Many(t *testing.T, newStore storeFactory) {
	mux := http.NewServeMux()
	mux.HandleFunc("/set", func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		sessionA := sessions.DefaultMany(ctx, "a")
		sessionA.Set("hello", "world")
		_ = sessionA.Save()

		sessionB := sessions.DefaultMany(ctx, "b")
		sessionB.Set("foo", "bar")
		_ = sessionB.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})

	mux.HandleFunc("/get", func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		sessionA := sessions.DefaultMany(ctx, "a")
		if sessionA.Get("hello") != "world" {
			t.Error("Session writing failed")
		}
		_ = sessionA.Save()

		sessionB := sessions.DefaultMany(ctx, "b")
		if sessionB.Get("foo") != "bar" {
			t.Error("Session writing failed")
		}
		_ = sessionB.Save()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ok))
	})

	sessionNames := []string{"a", "b"}

	handler := sessions.SessionsMany(mux, sessionNames, newStore(t, nil))

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set", nil)
	handler.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/get", nil)
	header := ""
	for _, x := range res1.Header()["Set-Cookie"] {
		header += strings.Split(x, ";")[0] + "; \n"
	}
	req2.Header.Set("Cookie", header)
	handler.ServeHTTP(res2, req2)
}

func copyCookies(req *http.Request, res *httptest.ResponseRecorder) {
	req.Header.Set("Cookie", strings.Join(res.Header().Values("Set-Cookie"), "; "))
}
