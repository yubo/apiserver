package register

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authorization.LOGIN"
)

// loginAuthorizer is an implementation of authorizer.Attributes
// which always says yes to an authorization request.
// It is useful in tests and when using kubernetes in an open manner.
type loginAuthorizer struct{}

func (loginAuthorizer) Authorize(ctx context.Context, a authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) {
	for _, group := range a.GetUser().GetGroups() {
		if group == user.AllAuthenticated {
			klog.V(5).Info("login allow")
			return authorizer.DecisionAllow, "Authorized", nil
		}
	}
	return authorizer.DecisionNoOpinion, "Unauthorized", nil
}

func (loginAuthorizer) RulesFor(userInfo user.Info, namespace string) ([]authorizer.ResourceRuleInfo, []authorizer.NonResourceRuleInfo, bool, error) {
	for _, group := range userInfo.GetGroups() {
		if group == user.AllAuthenticated {
			return []authorizer.ResourceRuleInfo{
					&authorizer.DefaultResourceRuleInfo{
						Verbs:     []string{"*"},
						Resources: []string{"*"},
					},
				}, []authorizer.NonResourceRuleInfo{
					&authorizer.DefaultNonResourceRuleInfo{
						Verbs:           []string{"*"},
						NonResourceURLs: []string{"*"},
					},
				}, false, nil
		}
	}
	return []authorizer.ResourceRuleInfo{}, []authorizer.NonResourceRuleInfo{}, false, nil
}

func init() {
	factory := func() (authorizer.Authorizer, error) {
		return &loginAuthorizer{}, nil
	}
	if err := authorization.RegisterAuthz("Login", factory); err != nil {
		panic(err)
	}
}
