package passwordfile

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	authUser "github.com/yubo/apiserver/pkg/authentication/user"
	"k8s.io/klog/v2"
)

type PasswordfileAuthenticator struct {
	users map[userPass]*authUser.DefaultInfo
}

type userPass struct {
	user     string
	password string
}

// NewCSV returns a PasswordAuthenticator, populated from a CSV file.
// The CSV file must contain records in the format "password,username,useruid"
func NewCSV(path string) (*PasswordfileAuthenticator, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	recordNum := 0
	users := make(map[userPass]*authUser.DefaultInfo)
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 3 {
			return nil, fmt.Errorf("password file '%s' must have at least 3 columns (password, user name, user uid), found %d", path, len(record))
		}

		recordNum++
		if record[0] == "" {
			klog.Warningf("empty password has been found in password file '%s', record number '%d'", path, recordNum)
			continue
		}

		obj := &authUser.DefaultInfo{
			Name: record[1],
			UID:  record[2],
		}

		key := userPass{user: record[1], password: record[0]}
		if _, exist := users[key]; exist {
			klog.Warningf("duplicate password has been found in password file '%s', record number '%d'", path, recordNum)
		}
		users[key] = obj

		if len(record) >= 4 {
			obj.Groups = strings.Split(record[3], ",")
		}
	}

	return &PasswordfileAuthenticator{users: users}, nil
}

func (a *PasswordfileAuthenticator) Authenticate(ctx context.Context, usr, pwd string) authUser.Info {
	if u, ok := a.users[userPass{user: usr, password: pwd}]; ok {
		return u
	}

	return nil
}

func (a *PasswordfileAuthenticator) Name() string {
	return "session authenticator"
}

func (a *PasswordfileAuthenticator) Priority() int {
	return authenticator.PRI_AUTH_PASSWORD
}

func (a *PasswordfileAuthenticator) Available() bool {
	return true
}
