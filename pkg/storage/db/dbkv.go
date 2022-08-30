package db

import (
	"context"
	"errors"
	"strings"

	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/runtime"
)

var _ storage.Store = &Store{}

// k8s.io/apiserver/pkg/registry/generic/registry/Store.go
type Store struct {
	db orm.Interface
}

func New(db orm.DB) *Store {
	return &Store{db: db}
}

// {table}/{namespace}/{name}
// {table}/{name}
// {table}
func parseKey(key string) (table, namespace, name string) {
	f := strings.Split(key, "/")
	if l := len(f); l >= 3 {
		return f[0], f[1], f[2]
	} else if l == 2 {
		return f[0], "", f[1]
	} else if l == 1 {
		return f[0], "", ""
	} else {
		return "", "", ""
	}
}

func parseKeyWithSelector(key, selector string) (string, string, error) {
	table, namespace, name := parseKey(key)

	if selector != "" {
		return table, selector, nil
	}

	if name == "" {
		return "", "", errors.New("key.name is empty")
	}

	q := "name=" + name

	if namespace != "" {
		q += ",namespace" + namespace
	}

	return table, q, nil
}

// AutoMigrate create table if not exist
func (p Store) AutoMigrate(ctx context.Context, key string, obj runtime.Object) error {
	table, _, _ := parseKey(key)

	return p.db.AutoMigrate(ctx, obj, orm.WithTable(table))
}

// drop table if exist
func (p Store) Drop(ctx context.Context, key string) error {
	table, _, _ := parseKey(key)

	opt, err := orm.NewOptions(orm.WithTable(table))
	if err != nil {
		return err
	}

	return p.db.DropTable(ctx, opt)
}

func (p Store) Create(ctx context.Context, key string, obj, out runtime.Object) error {
	table, selector, err := parseKeyWithSelector(key, "")
	if err != nil {
		return err
	}

	if err := p.db.Insert(ctx, obj, orm.WithTable(table)); err != nil {
		return err
	}

	if out == nil {
		return nil
	}

	return p.get(ctx, table, selector, false, out)
}

func (p Store) Delete(ctx context.Context, key string, out runtime.Object) error {
	table, selector, err := parseKeyWithSelector(key, "")
	if err != nil {
		return err
	}

	if out != nil {
		if err := p.get(ctx, table, selector, false, out); err != nil {
			return err
		}
	}

	return p.db.Delete(ctx, nil, orm.WithTable(table), orm.WithSelector(selector))
}

func (p Store) Update(ctx context.Context, key string, obj, out runtime.Object) error {
	table, selector, err := parseKeyWithSelector(key, "")
	if err != nil {
		return err
	}

	if err := p.db.Update(ctx, obj, orm.WithTable(table)); err != nil {
		return err
	}

	if out == nil {
		return nil
	}

	return p.get(ctx, table, selector, false, out)
}

func (p Store) Get(ctx context.Context, key string, opts storage.GetOptions, out runtime.Object) error {
	table, selector, err := parseKeyWithSelector(key, "")
	if err != nil {
		return err
	}

	return p.get(ctx, table, selector, opts.IgnoreNotFound, out)
}

func (p Store) get(ctx context.Context, table, selector string, ignoreNotFound bool, out runtime.Object) error {
	return p.db.Get(ctx, out, orm.WithTable(table), orm.WithSelector(selector), orm.WithIgnoreNotFoundErr(ignoreNotFound))
}

func (p Store) List(ctx context.Context, key string, opts storage.ListOptions, out runtime.Object, total *int) error {
	table, _, _ := parseKey(key)

	return p.db.List(
		ctx,
		out,
		orm.WithTable(table),
		orm.WithTotal(total),
		orm.WithSelector(opts.Query),
		orm.WithOrderby(opts.Orderby...),
		orm.WithLimit(opts.Offset, opts.Limit),
	)
}
