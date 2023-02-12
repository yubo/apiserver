package logging

import (
	"testing"

	v1 "github.com/yubo/apiserver/components/logs/api/v1"
	"github.com/yubo/golib/util"
)

func TestConfig(t *testing.T) {
	config := NewConfig()
	config.VModule = v1.VModuleConfiguration([]v1.VModuleItem{
		{
			FilePattern: "test",
			Verbosity:   10,
		}})
	t.Logf("config %s", util.JsonStr(config))
}
