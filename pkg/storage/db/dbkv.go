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

func (p Store) getdb(ctx context.Context) orm.Interface {
	// for transaction (tx)
	if db, ok := orm.InterfaceFrom(ctx); ok {
		return db
	}

	return p.db
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
func (p Store) AutoMigrate(key string, obj runtime.Object) error {
	db := p.getdb(context.TODO())
	table, _, _ := parseKey(key)

	return db.AutoMigrate(obj, orm.WithTable(table))
}

// drop table if exist
func (p Store) Drop(key string) error {
	db := p.getdb(context.TODO())
	table, _, _ := parseKey(key)

	opt, err := orm.NewOptions(orm.WithTable(table))
	if err != nil {
		return err
	}

	return db.DropTable(opt)
}

func (p Store) Create(ctx context.Context, key string, obj, out runtime.Object) error {
	db := p.getdb(ctx)

	table, selector, err := parseKeyWithSelector(key, "")
	if err != nil {
		return err
	}

	if err := db.Insert(obj, orm.WithTable(table)); err != nil {
		return err
	}

	if out == nil {
		return nil
	}

	return p.get(db, table, selector, false, out)
}

func (p Store) Delete(ctx context.Context, key string, out runtime.Object) error {
	db := p.getdb(ctx)

	table, selector, err := parseKeyWithSelector(key, "")
	if err != nil {
		return err
	}

	if out != nil {
		if err := p.get(db, table, selector, false, out); err != nil {
			return err
		}
	}

	return db.Delete(nil, orm.WithTable(table), orm.WithSelector(selector))
}

func (p Store) Update(ctx context.Context, key string, obj, out runtime.Object) error {
	db := p.getdb(ctx)

	table, selector, err := parseKeyWithSelector(key, "")
	if err != nil {
		return err
	}

	if err := db.Update(obj, orm.WithTable(table)); err != nil {
		return err
	}

	if out == nil {
		return nil
	}

	return p.get(db, table, selector, false, out)
}

func (p Store) Get(ctx context.Context, key string, opts storage.GetOptions, out runtime.Object) error {
	db := p.getdb(ctx)

	table, selector, err := parseKeyWithSelector(key, "")
	if err != nil {
		return err
	}

	return p.get(db, table, selector, opts.IgnoreNotFound, out)
}

func (p Store) get(db orm.Interface, table, selector string, ignoreNotFound bool, out runtime.Object) error {
	opts := []orm.Option{orm.WithTable(table), orm.WithSelector(selector)}
	if ignoreNotFound {
		opts = append(opts, orm.WithIgnoreNotFoundErr())
	}

	return db.Get(out, opts...)
}

func (p Store) List(ctx context.Context, key string, opts storage.ListOptions, out runtime.Object, total *int64) error {
	db := p.getdb(ctx)
	table, _, _ := parseKey(key)

	return db.List(
		out,
		orm.WithTable(table),
		orm.WithTotal(total),
		orm.WithSelector(opts.Query),
		orm.WithOrderby(opts.Orderby...),
		orm.WithLimit(opts.Offset, opts.Limit),
	)
}
