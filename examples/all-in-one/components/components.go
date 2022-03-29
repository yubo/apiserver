package components

import (
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
	// 3.session
	_ "github.com/yubo/apiserver/plugin/authenticator/session/register"
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
	_ "github.com/yubo/apiserver/pkg/traces/register"
	_ "github.com/yubo/apiserver/pkg/version/reporter/register"
)
