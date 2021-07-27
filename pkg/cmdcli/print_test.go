// Copyright 2018,2019 freewheel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cmdcli

import "testing"

func TestPreetyTab(t *testing.T) {
	in := `https://arti.dev.fwmrm.net	a.json no cached
https://arti.rnd.fwmrm.net	a.json invalidated
https://arti-dev.ss.aws.fwmrm.net	a.json invalidated
https://arti.stg.fwmrm.net	a.json invalidated
https://arti-stg.ss.aws.fwmrm.net	a.json invalidated
https://arti.fwmrm.net	a.json invalidated
https://arti-ams.fwmrm.net	a.json invalidated
https://arti-svl.fwmrm.net	a.json invalidated
https://arti.ss.aws.fwmrm.net	a.json invalidated`
	out := PrettyTab(in)
	t.Log(out)
}
