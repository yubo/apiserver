package auth

import (
	// Initialize all known client auth plugins.
	_ "github.com/yubo/apiserver/pkg/client/auth/azure"
	_ "github.com/yubo/apiserver/pkg/client/auth/gcp"
	_ "github.com/yubo/apiserver/pkg/client/auth/oidc"
	_ "github.com/yubo/apiserver/pkg/client/auth/openstack"
)
