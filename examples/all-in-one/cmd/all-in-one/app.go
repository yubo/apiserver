package main

import (
	"context"
	"examples/all-in-one/pkg/allinone"

	"github.com/go-openapi/spec"
	"github.com/spf13/cobra"
	"github.com/yubo/apiserver/components/version"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"

	_ "github.com/yubo/apiserver/components/logs/json/register"

	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"

	// authz's submodule, should be loaded before the authz module
	_ "github.com/yubo/apiserver/pkg/authorization/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/abac/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/alwaysallow/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/alwaysdeny/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/rbac/register"

	// TODO
	//_ "github.com/yubo/apiserver/plugin/authorizer/webhook/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"

	// 1. headerrequest
	_ "github.com/yubo/apiserver/plugin/authenticator/headerrequest/register"
	// 2. x509
	_ "github.com/yubo/apiserver/plugin/authenticator/x509/register"
	// 3.session deprecated
	//_ "github.com/yubo/apiserver/plugin/authenticator/session/register"
	// 4. tokenfile
	_ "github.com/yubo/apiserver/plugin/authenticator/token/tokenfile/register"
	// 5. service account file <TODO>
	// 6. service account issuer <TODO>
	// 7. bootstrap
	_ "github.com/yubo/apiserver/plugin/authenticator/token/bootstrap/register"
	// 8. OIDC
	_ "github.com/yubo/apiserver/plugin/authenticator/token/oidc/register"
	// 9. webhook
	_ "github.com/yubo/apiserver/plugin/authenticator/token/webhook/register"

	_ "github.com/yubo/apiserver/pkg/audit/register"
	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/apiserver/pkg/grpcserver/register"
	_ "github.com/yubo/apiserver/pkg/models/register"
	_ "github.com/yubo/apiserver/pkg/server/register"
	_ "github.com/yubo/apiserver/pkg/session/register"
	_ "github.com/yubo/apiserver/pkg/tracing/register"
)

const (
	moduleName = "all-in-one.example.apiserver"
)

type config struct {
	TestA string `json:"testA" flag:"test-a" description:"this is a flag for demo"`
	TestB string `json:"testB"`
}

func newConfig() *config {
	return &config{}
}

var (
	hookOps = []v1.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  v1.ACTION_START,
		Priority: v1.PRI_MODULE,
	}}
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

func newServerCmd() *cobra.Command {
	cmd := proc.NewRootCmd(
		proc.WithHooks(hookOps...),
		proc.WithName("all-in-one"),
		proc.WithDescription("apiserver examples all in one"),
		proc.WithVersion(version.Get()),
		proc.WithLicense(&license),
		proc.WithContact(&contact),
		proc.WithReport(),
	)
	//cmd.AddCommand(proc.NewVersionCmd())

	return cmd
}

func start(ctx context.Context) error {
	if err := allinone.New().Start(ctx); err != nil {
		return err
	}

	return nil
}

func init() {
	proc.AddGlobalConfig(newConfig())
}
