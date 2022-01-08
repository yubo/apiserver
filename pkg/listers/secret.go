package listers

import (
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/queries"
)

// SecretLister helps list Secrets.
// All objects returned here must be treated as read-only.
type SecretLister interface {
	// List lists all Secrets in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector queries.Selector) (ret []*api.Secret, err error)
	// Get retrieves the Secret from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*api.Secret, error)
}

// secretLister implements the SecretLister interface.
//type secretLister struct {
//	db orm.DB
//}
//
//// NewSecretLister returns a new SecretLister.
//func NewSecretLister(db orm.DB) SecretLister {
//	return &secretLister{db: db}
//}
//
//// List lists all Secrets in the indexer.
//func (s *secretLister) List(selector queries.Selector) (ret []*api.Secret, err error) {
//	err = storage.List(s.db, "secret", selector, &ret)
//	return
//}
//
//func (s *secretLister) Get(name string) (ret *api.Secret, err error) {
//	err = storage.Get(s.db, "secret", name, &ret)
//	return
//}
