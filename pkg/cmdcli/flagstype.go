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
