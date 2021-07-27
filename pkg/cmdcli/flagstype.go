// Copyright 2018,2019 freewheel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cmdcli

import (
	"fmt"

	"github.com/yubo/golib/util"
)

type Seconds int64

func (p *Seconds) Set(s string) error {
	*p = Seconds(util.TimeOf(s))
	return nil
}

func (p *Seconds) String() string {
	return fmt.Sprintf("%d", *p)
}

func (p *Seconds) Type() string {
	return "Seconds"
}
