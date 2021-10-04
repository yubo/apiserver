package storage

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yubo/golib/labels"
	"github.com/yubo/golib/orm"
)

const (
	MAX_ROWS = 200
)

func List(db *orm.DB, kind string, selector labels.Selector, dst interface{}) error {
	return nil
}

func Get(db *orm.DB, kind string, name string, dst interface{}) error {
	return nil
}

type ListerOptions struct {
	DB      orm.DB
	Table   string
	Index   string
	Limit   int64
	Offset  int64
	Columns []string
	Orderby []string
	Querys  []string
	Args    []interface{}
}

type Lister struct {
	db      orm.DB
	table   string
	index   string
	limit   int64
	offset  int64
	columns []string
	orderby []string
	querys  []string
	args    []interface{}
}

func (in *Lister) clone() (out *Lister) {
	*out = *in
	if in.columns != nil {
		in, out := &in.args, &out.args
		*out = make([]interface{}, len(*in))
		copy(*out, *in)
	}
	if in.orderby != nil {
		in, out := &in.orderby, &out.orderby
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.querys != nil {
		in, out := &in.querys, &out.querys
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.args != nil {
		in, out := &in.args, &out.args
		*out = make([]interface{}, len(*in))
		copy(*out, *in)
	}
	return
}

func NewLister(opts ListerOptions) *Lister {
	out := &Lister{
		db:      opts.DB,
		table:   opts.Table,
		index:   opts.Index,
		columns: opts.Columns,
		limit:   opts.Limit,
		offset:  opts.Offset,
		orderby: opts.Orderby,
		querys:  opts.Querys,
		args:    opts.Args,
	}

	if out.limit <= 0 || out.limit > MAX_ROWS {
		out.limit = MAX_ROWS
	}
	return out
}

func (p *Lister) Options(opts ListerOptions) *Lister {
	out := p.clone()
	if opts.DB != nil {
		out.db = opts.DB
	}
	if opts.Limit > 0 {
		out.limit = opts.Limit
	}
	if opts.Offset > 0 {
		out.offset = opts.Offset
	}
	if len(opts.Columns) > 0 {
		out.columns = opts.Columns

	}
	if len(opts.Orderby) > 0 {
		out.orderby = append(out.orderby, opts.Orderby...)
	}
	if len(opts.Querys) > 0 {
		out.querys = append(out.querys, opts.Querys...)
	}
	if len(opts.Args) > 0 {
		out.args = append(out.args, opts.Args...)
	}

	return out
}

func (p *Lister) Where(query string, args ...interface{}) *Lister {
	out := p.clone()
	out.querys = append(out.querys, query)
	out.args = append(out.args, args...)
	return out
}

func (p *Lister) Limit(limit, offset int64) *Lister {
	out := p.clone()
	out.limit = limit
	out.offset = offset
	return out
}

func (p *Lister) Orderby(orderby string) *Lister {
	out := p.clone()
	out.orderby = append(out.orderby, orderby)
	return out
}

func (p *Lister) List() *orm.Rows {
	buf := &bytes.Buffer{}

	buf.WriteString("select " + p.table + " from ")

	if len(p.columns) > 0 {
		buf.WriteString(strings.Join(p.columns, ", ") + " ")
	} else {
		buf.WriteString("* ")
	}

	if l := len(p.querys); l == 1 {
		buf.WriteString("where " + p.querys[0] + " ")
	} else if l > 1 {
		buf.WriteString("where (" + strings.Join(p.querys, ") and (") + ") ")
	}

	if len(p.orderby) > 0 {
		buf.WriteString("order by " + strings.Join(p.orderby, ", ") + " ")
	}

	if p.limit > 0 {
		fmt.Fprintf(buf, "limit %d, %d", p.offset, p.limit)
	}

	return p.clone().db.Query(buf.String(), p.args...)
}

func (p *Lister) Get(name string) *orm.Rows {
	var index string
	if index = p.index; index == "" {
		index = "name"
	}

	buf := &bytes.Buffer{}
	buf.WriteString("select " + p.table + " from ")

	if len(p.columns) > 0 {
		buf.WriteString(strings.Join(p.columns, ", ") + " ")
	} else {
		buf.WriteString("* ")
	}

	if len(p.index) > 0 {
		fmt.Fprintf(buf, "where %s = ?", p.index)
	} else {
		buf.WriteString("where name = ?")
	}

	return p.clone().db.Query(buf.String(), name)
}
