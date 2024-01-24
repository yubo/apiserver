package main

import (
	"context"
	"examples/all-in-one/pkg/allinone"

	"github.com/go-openapi/spec"
	"github.com/spf13/cobra"
	"github.com/yubo/apiserver/components/version"
	"github.com/yubo/apiserver/pkg/proc"

	// log format
	_ "github.com/yubo/apiserver/components/logs/json/register"

	// db
	"github.com/yubo/golib/orm"
	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"

	// authn plugin
	_ "github.com/yubo/apiserver/plugin/authenticator/passwordfile/register"
	_ "github.com/yubo/apiserver/plugin/authenticator/session/register"

	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/apiserver/pkg/grpcserver/register"
	_ "github.com/yubo/apiserver/pkg/server/register"
	_ "github.com/yubo/apiserver/pkg/sessions/cookie"
	_ "github.com/yubo/apiserver/pkg/sessions/orm"
	_ "github.com/yubo/apiserver/pkg/sessions/register"
	_ "github.com/yubo/apiserver/pkg/tracing/register"
)

var (
	license = spec.License{
		LicenseProps: spec.LicenseProps{
			Name: "Apache-2.0",
			URL:  "https://www.apache.org/licenses/LICENSE-2.0.txt",
		},
	}
	contact = spec.ContactInfo{
		ContactInfoProps: spec.ContactInfoProps{
			Name:  "yubo",
			URL:   "http://github.com/yubo",
			Email: "yubo@yubo.org",
		},
	}
)

func init() {
	orm.DEBUG = true
}

func newServerCmd() *cobra.Command {
	return proc.NewRootCmd(
		proc.WithRun(start),
		proc.WithName("all-in-one"),
		proc.WithDescription("apiserver examples all in one"),
		proc.WithVersion(version.Get()),
		proc.WithLicense(&license),
		proc.WithContact(&contact),
		proc.WithReport(),
	)
}

func start(ctx context.Context) error {
	if err := allinone.New().Start(ctx); err != nil {
		return err
	}

	return nil
}
