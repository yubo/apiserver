package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// flags:long-name<,short-name>,defualt-value
func TestGetArgs(t *testing.T) {
	type Foo struct {
		A string `flag:"-" json:",arg1"`
		B string `flag:"b-name"`
	}
	cases := []struct {
		in   Foo
		want []string
	}{
		{Foo{A: "a1", B: "b1"}, []string{"a1", "--b-name", "b1"}},
	}

	for i, c := range cases {
		got := []string{}
		err := GetArgs(&got, nil, c.in)
		require.Emptyf(t, err, "case-%d", i)
		require.Equalf(t, c.want, got, "case-%d", i)
	}
}
