// Copyright 2018,2019 freewheel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cmdcli

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yubo/golib/encoding/postfile"
)

// flags:long-name<,short-name>,defualt-value
func TestGetArgs(t *testing.T) {
	type Foo struct {
		A string            `flags:",arg"`
		B string            `flags:"b-name,,"`
		C int               `flags:"c-name,,"`
		D uint              `flags:"d-name,,"`
		E []string          `flags:"e-name,,"`
		F postfile.PostFile `flags:"f-name,,"`
	}
	type Bar struct {
		Foo `flags:",inline"`
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
