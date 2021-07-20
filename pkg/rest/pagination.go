package rest

import (
	"fmt"
	"strings"

	"github.com/yubo/golib/util"
)

var (
	maxLimitPage = 500
	defLimitPage = 10
)

func GetLimit(limit int) int {
	if limit <= 0 {
		return defLimitPage
	}

	if limit > maxLimitPage {
		return maxLimitPage
	}

	return limit
}

func SetLimitPage(def, max int) {
	if def > 0 {
		defLimitPage = def
	}
	if max > 0 {
		maxLimitPage = max
	}
}

type Pagination struct {
	Offset      int     `param:"query,hidden" flags:"-" description:"offset, priority is more than currentPage"`
	Limit       int     `param:"query,hidden" flags:"-" description:"limit", priority is more than pageSize`
	PageSize    int     `param:"query" flags:"-" description:"page size"`
	CurrentPage int     `param:"query" flags:"-" description:"current page number, start at 1(defualt)"`
	Sorter      *string `param:"query" flags:"-" description:"column name"`
	Order       *string `param:"query" flags:"-" description:"asc(default)/desc"`
}

func (p *Pagination) OffsetLimit() (offset, limit int) {
	limit = p.Limit

	if limit == 0 {
		limit = p.PageSize
	}

	if limit == 0 {
		limit = defLimitPage
	}

	if limit > maxLimitPage {
		limit = maxLimitPage
	}

	offset = p.Offset

	if offset <= 0 {
		offset = (p.CurrentPage - 1) * limit
	}

	if offset < 0 {
		offset = 0
	}

	return
}

func (p Pagination) SqlExtra(orders ...string) string {
	offset, limit := p.OffsetLimit()

	var order string
	if sorter := util.SnakeCasedName(util.StringValue(p.Sorter)); sorter != "" {
		orders = append([]string{"`" + sorter + "` " +
			sqlOrder(util.StringValue(p.Order))}, orders...)
	}

	if len(orders) > 0 {
		order = " order by " + strings.Join(orders, ", ")
	}

	return fmt.Sprintf(order+" limit %d, %d", offset, limit)
}

// ungly hack
func (p Pagination) SqlExtra2(prefix string, orders ...string) string {
	offset, limit := p.OffsetLimit()

	var order string
	if sorter := util.SnakeCasedName(util.StringValue(p.Sorter)); sorter != "" {
		orders = append([]string{fmt.Sprintf("`%s.%s` %s",
			prefix, sorter, sqlOrder(util.StringValue(p.Order)))},
			orders...)
	}

	if len(orders) > 0 {
		order = " order by " + strings.Join(orders, ", ")
	}

	return fmt.Sprintf(order+" limit %d, %d", offset, limit)
}

func sqlOrder(order string) string {
	switch strings.ToLower(order) {
	case "ascend", "asc":
		return "asc"
	case "descend", "desc":
		return "desc"
	default:
		return "asc"
	}
}
