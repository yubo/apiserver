package oidc

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/token/oidc"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	pconfig "github.com/yubo/golib/proc/config"
	cliflag "github.com/yubo/golib/staging/cli/flag"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

const (
	moduleName       = "authentication.oidc"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _auth.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT - 1,
		SubPriority: options.PRI_M_AUTHN,
	}}
	_config *config
)

type config struct {
	CAFile         string
	ClientID       string
	IssuerURL      string
	UsernameClaim  string
	UsernamePrefix string
	GroupsClaim    string
	GroupsPrefix   string
	SigningAlgs    []string
	RequiredClaims map[string]string
}

func (o *config) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.IssuerURL, "oidc-issuer-url", o.IssuerURL, ""+
		"The URL of the OpenID issuer, only HTTPS scheme will be accepted. "+
		"If set, it will be used to verify the OIDC JSON Web Token (JWT).")

	fs.StringVar(&o.ClientID, "oidc-client-id", o.ClientID,
		"The client ID for the OpenID Connect client, must be set if oidc-issuer-url is set.")

	fs.StringVar(&o.CAFile, "oidc-ca-file", o.CAFile, ""+
		"If set, the OpenID server's certificate will be verified by one of the authorities "+
		"in the oidc-ca-file, otherwise the host's root CA set will be used.")

	fs.StringVar(&o.UsernameClaim, "oidc-username-claim", "sub", ""+
		"The OpenID claim to use as the user name. Note that claims other than the default ('sub') "+
		"is not guaranteed to be unique and immutable. This flag is experimental, please see "+
		"the authentication documentation for further details.")

	fs.StringVar(&o.UsernamePrefix, "oidc-username-prefix", "", ""+
		"If provided, all usernames will be prefixed with this value. If not provided, "+
		"username claims other than 'email' are prefixed by the issuer URL to avoid "+
		"clashes. To skip any prefixing, provide the value '-'.")

	fs.StringVar(&o.GroupsClaim, "oidc-groups-claim", "", ""+
		"If provided, the name of a custom OpenID Connect claim for specifying user groups. "+
		"The claim value is expected to be a string or array of strings. This flag is experimental, "+
		"please see the authentication documentation for further details.")

	fs.StringVar(&o.GroupsPrefix, "oidc-groups-prefix", "", ""+
		"If provided, all groups will be prefixed with this value to prevent conflicts with "+
		"other authentication strategies.")

	fs.StringSliceVar(&o.SigningAlgs, "oidc-signing-algs", []string{"RS256"}, ""+
		"Comma-separated list of allowed JOSE asymmetric signing algorithms. JWTs with a "+
		"'alg' header value not in this list will be rejected. "+
		"Values are defined by RFC 7518 https://tools.ietf.org/html/rfc7518#section-3.1.")

	fs.Var(cliflag.NewMapStringStringNoSplit(&o.RequiredClaims), "oidc-required-claim", ""+
		"A key=value pair that describes a required claim in the ID Token. "+
		"If set, the claim is verified to be present in the ID Token with a matching value. "+
		"Repeat this flag to specify multiple claims.")

}

func (o *config) changed() interface{} {
	if o == nil {
		return nil
	}
	return util.Diff2Map(defaultConfig(), o)
}

func (o *config) Validate() error {
	if (len(o.IssuerURL) > 0) != (len(o.ClientID) > 0) {
		return fmt.Errorf("oidc-issuer-url and oidc-client-id should be specified together")
	}

	if o.UsernamePrefix == "" && o.UsernameClaim != "email" {
		// Old behavior. If a usernamePrefix isn't provided, prefix all claims other than "email"
		// with the issuerURL.
		//
		// See https://github.com/kubernetes/kubernetes/issues/31380
		o.UsernamePrefix = o.IssuerURL + "#"
	}

	if o.UsernamePrefix == noUsernamePrefix {
		// Special value indicating usernames shouldn't be prefixed.
		o.UsernamePrefix = ""
	}

	return nil
}

type authModule struct {
	name   string
	config *config
}

func defaultConfig() *config {
	return &config{}
}

func (p *authModule) init(ops *proc.HookOps) error {
	configer := ops.Configer()

	cf := defaultConfig()
	if err := configer.ReadYaml(p.name, cf,
		pconfig.WithOverride(_config.changed())); err != nil {
		return err
	}
	p.config = cf

	if len(cf.IssuerURL) == 0 {
		klog.Infof("%s is not set, skip", p.name)
		return nil
	}

	auth, err := oidc.New(oidc.Options{
		IssuerURL:            cf.IssuerURL,
		ClientID:             cf.ClientID,
		CAFile:               cf.CAFile,
		UsernameClaim:        cf.UsernameClaim,
		UsernamePrefix:       cf.UsernamePrefix,
		GroupsClaim:          cf.GroupsClaim,
		GroupsPrefix:         cf.GroupsPrefix,
		SupportedSigningAlgs: cf.SigningAlgs,
		RequiredClaims:       cf.RequiredClaims,
	})
	if err != nil {
		return err
	}

	return authentication.RegisterTokenAuthn(auth)
}

func init() {
	proc.RegisterHooks(hookOps)
	_config = defaultConfig()
	_config.addFlags(proc.NamedFlagSets().FlagSet("authentication"))
}
