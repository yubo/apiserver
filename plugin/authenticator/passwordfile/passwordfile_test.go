package passwordfile

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yubo/apiserver/pkg/authentication/user"
)

func TestPasswordFile(t *testing.T) {
	auth, err := newWithContents(t, `
password1,user1,uid1
password2,user2,uid2
password3,user3,uid3,"group1,group2"
password4,user4,uid4,"group2"
password5,user5,uid5,group5
password6,user6,uid6,group5,otherdata
password7,user7,uid7,"group1,group2",otherdata
`)
	if err != nil {
		t.Fatalf("unable to read passwordfile: %v", err)
	}

	testCases := []struct {
		Password string
		User     *user.DefaultInfo
		Ok       bool
	}{
		{
			Password: "password1",
			User:     &user.DefaultInfo{Name: "user1", UID: "uid1"},
			Ok:       true,
		},
		{
			Password: "password2",
			User:     &user.DefaultInfo{Name: "user2", UID: "uid2"},
			Ok:       true,
		},
		{
			Password: "password3",
			User:     &user.DefaultInfo{Name: "user3", UID: "uid3", Groups: []string{"group1", "group2"}},
			Ok:       true,
		},
		{
			Password: "password4",
			User:     &user.DefaultInfo{Name: "user4", UID: "uid4", Groups: []string{"group2"}},
			Ok:       true,
		},
		{
			Password: "password5",
			User:     &user.DefaultInfo{Name: "user5", UID: "uid5", Groups: []string{"group5"}},
			Ok:       true,
		},
		{
			Password: "password6",
			User:     &user.DefaultInfo{Name: "user6", UID: "uid6", Groups: []string{"group5"}},
			Ok:       true,
		},
		{
			Password: "password7",
			User:     &user.DefaultInfo{Name: "user7", UID: "uid7", Groups: []string{"group1", "group2"}},
			Ok:       true,
		},
		{
			Password: "password8",
		},
	}
	for i, testCase := range testCases {
		var name string
		if testCase.User != nil {
			name = testCase.User.Name
		}
		resp := auth.Authenticate(context.Background(), name, testCase.Password)
		if testCase.User == nil {
			assert.Nil(t, resp, i)
		} else {
			assert.Equal(t, testCase.User, resp, i)
		}
		assert.Equal(t, testCase.Ok, resp != nil, i)
	}
}

func TestBadPasswordFile(t *testing.T) {
	_, err := newWithContents(t, `
password1,user1,uid1
password2,user2,uid2
password3,user3
password4
`)
	if err == nil {
		t.Fatalf("unexpected non error")
	}
}

func TestInsufficientColumnspasswordFile(t *testing.T) {
	_, err := newWithContents(t, "password4\n")
	assert.Error(t, err)
}

func TestEmptyPasswordPasswordFile(t *testing.T) {
	auth, err := newWithContents(t, ",user5,uid5\n")
	assert.NoError(t, err)
	assert.Len(t, auth.users, 0, "empty password should not be recorded")
}

func newWithContents(t *testing.T, contents string) (auth *PasswordfileAuthenticator, err error) {
	f, err := ioutil.TempFile("", "passwordfile_test")
	if err != nil {
		t.Fatalf("unexpected error creating passwordfile: %v", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if err := ioutil.WriteFile(f.Name(), []byte(contents), 0700); err != nil {
		t.Fatalf("unexpected error writing passwordfile: %v", err)
	}

	return NewCSV(f.Name())
}
