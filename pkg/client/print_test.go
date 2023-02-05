package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableStr(t *testing.T) {
	var user = []struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	}{
		{"user1", "123"},
		{"user2", "45678"},
	}
	assert.Equal(t, `Name   Phone
user1  123
user2  45678
`, TableStr(user))
}
