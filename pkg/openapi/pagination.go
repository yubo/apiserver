package openapi

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
	PageSize    *int    `param:"query" flags:"-" description:"page size"`
	CurrentPage *int    `param:"query" flags:"-" description:"current page number, start at 1(defualt)"`
	Sorter      *string `param:"query" flags:"-" description:"column name"`
	Order       *string `param:"query" flags:"-" description:"asc(default)/desc"`
}

func (p *Pagination) OffsetLimit() (int, int) {
	limit := util.IntValue(p.PageSize)

	if limit == 0 {
		limit = defLimitPage
	}

	if limit > maxLimitPage {
		limit = maxLimitPage
	}

	currentPage := util.IntValue(p.CurrentPage)
	if currentPage <= 1 {
		return 0, limit
	}

	return (currentPage - 1) * limit, limit
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
	switch order {
	case "ascend", "asc":
		return "asc"
	case "descend", "desc":
		return "desc"
	default:
		return "asc"
	}
}
