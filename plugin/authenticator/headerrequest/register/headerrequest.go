package headerrequest

import (
	"context"
	"fmt"
	"strings"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/request/headerrequest"
	"github.com/yubo/apiserver/pkg/dynamiccertificates"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/golib/util/errors"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authentication.requestheader"
	configPath = "authentication.requestheader"
)

type config struct {
	// ClientCAFile is the root certificate bundle to verify client certificates on incoming requests
	// before trusting usernames in headers.
	ClientCAFile string `json:"clientCAFile" flag:"requestheader-client-ca-file" description:"Root certificate bundle to use to verify client certificates on incoming requests before trusting usernames in headers specified by --requestheader-username-headers. WARNING: generally do not depend on authorization being already done for incoming requests."`

	UsernameHeaders     []string `json:"usernameHeaders" flag:"requestheader-username-headers" description:"List of request headers to inspect for usernames. X-Remote-User is common."`
	GroupHeaders        []string `json:"groupHeaders" flag:"requestheader-group-headers" description:"List of request headers to inspect for groups. X-Remote-Group is suggested."`
	ExtraHeaderPrefixes []string `json:"extraHeaderPrefixes" flag:"requestheader-extra-headers-prefix" description:"List of request header prefixes to inspect. X-Remote-Extra- is suggested."`
	AllowedNames        []string `json:"allowedNames" flag:"requestheader-allowed-names" description:"List of client certificate common names to allow to provide usernames in headers specified by --requestheader-username-headers. If empty, any client certificate validated by the authorities in --requestheader-client-ca-file is allowed."`
}

func (s *config) Validate() error {
	allErrors := []error{}

	if err := checkForWhiteSpaceOnly("requestheader-username-headers", s.UsernameHeaders...); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := checkForWhiteSpaceOnly("requestheader-group-headers", s.GroupHeaders...); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := checkForWhiteSpaceOnly("requestheader-extra-headers-prefix", s.ExtraHeaderPrefixes...); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := checkForWhiteSpaceOnly("requestheader-allowed-names", s.AllowedNames...); err != nil {
		allErrors = append(allErrors, err)
	}

	return errors.NewAggregate(allErrors)
}

func checkForWhiteSpaceOnly(flag string, headerNames ...string) error {
	for _, headerName := range headerNames {
		if len(strings.TrimSpace(headerName)) == 0 {
			return fmt.Errorf("empty value in %q", flag)
		}
	}

	return nil
}
func newConfig() *config { return &config{} }

func factory(ctx context.Context) (authenticator.Request, error) {
	cf := newConfig()
	if err := proc.ReadConfig(configPath, cf); err != nil {
		return nil, err
	}

	if cf.ClientCAFile == "" {
		klog.V(5).Infof("authnModule %s clientCAFile is not set, ignore", moduleName)
		return nil, nil
	}

	klog.V(5).InfoS("authnModule init", "name", moduleName)

	caBundleProvider, err := dynamiccertificates.NewDynamicCAContentFromFile("request-header", cf.ClientCAFile)
	if err != nil {
		return nil, err
	}

	return authenticator.WrapAudienceAgnosticRequest(
		authentication.APIAudiences(), headerrequest.NewDynamicVerifyOptionsSecure(
			caBundleProvider.VerifyOptions,
			headerrequest.StaticStringSlice(cf.AllowedNames),
			headerrequest.StaticStringSlice(cf.UsernameHeaders),
			headerrequest.StaticStringSlice(cf.GroupHeaders),
			headerrequest.StaticStringSlice(cf.ExtraHeaderPrefixes),
		)), nil

}

func init() {
	authentication.RegisterAuthn(factory)
	proc.AddConfig(configPath, newConfig(), proc.WithConfigGroup("authentication"))
}
