package cmdcli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strings"

	"github.com/buger/goterm"
	"github.com/yubo/apiserver/pkg/rest"
	apierrors "github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util/term"
	"github.com/yubo/golib/util"
)

type TermPager struct {
	pagination *rest.Pagination
	disable    bool
	buff       []byte
	out        io.Writer
	pageTotal  int // start from 1
	total      int
	method     string
	uri        string
	input      interface{}
	output     interface{}
	cb         []func(interface{})
	handle     func(method, uri string, input, output interface{},
		cb ...func(interface{})) error
}

func (p *TermPager) FootBarRender(format string, a ...interface{}) {
	extra := fmt.Sprintf(format, a...)

	fmt.Fprintf(p.out, "\r%s %s\033[K",
		goterm.Color(fmt.Sprintf("%s/%d", string(p.buff), p.pageTotal),
			goterm.GREEN), extra)
}

func (p *TermPager) Render(page int, rerend bool) (err error) {
	defer func() {
		if err == nil {
			p.buff = []byte(fmt.Sprintf("%d", page))
			p.FootBarRender("")
		} else {
			p.FootBarRender(goterm.Color(err.Error(), goterm.RED))
		}
	}()

	if page > p.pageTotal || page < 1 {
		err = fmt.Errorf("page valid range [1,%d], got %d",
			p.pageTotal, page)
		return
	}

	p.pagination.CurrentPage = page

	// send query
	if err = p.handle("GET", p.uri, p.input, p.output, p.cb...); err != nil {
		return
	}

	pageSize := p.pagination.PageSize
	if rerend {
		fmt.Fprintf(p.out, "\033[%dA\r", pageSize+1)
	}

	fmt.Fprintf(p.out, strings.Replace(string(Table(p.output)),
		"\n", "\033[K\n", -1))

	if v := reflect.Indirect(reflect.ValueOf(p.output)); v.Kind() ==
		reflect.Slice || v.Kind() == reflect.Array {
		if n := pageSize - v.Len(); n > 0 {
			fmt.Fprintf(p.out, strings.Repeat("\033[K\n", n))
		}
	}

	return
}

func (p *TermPager) Dump() (err error) {
	totalPage := p.total / p.pagination.PageSize

	for i := 0; i < totalPage; i++ {
		p.pagination.CurrentPage = i
		if err = p.handle("GET", p.uri, p.input, p.output,
			p.cb...); err != nil {
			return
		}
		output := Table(p.output)
		if i > 0 {
			if i := bytes.IndexByte(output, '\n'); i > 0 {
				output = output[i+1:]
			}
		}
		p.out.Write(output)
	}
	return nil
}

func (p *TermPager) Run() error {
	if p.pagination.Dump {
		return p.Dump()
	}

	defer func() {
		// Show cursor.
		fmt.Fprintf(p.out, "\033[?25h\n")
	}()

	pageSize := p.pagination.PageSize
	p.pageTotal = int(math.Ceil(float64(p.total) / float64(pageSize)))

	p.Render(p.pagination.CurrentPage, false)

	// Hide cursor.
	fmt.Fprintf(p.out, "\033[?25l")

	for {
		ascii, keyCode, err := term.Getch()
		if err != nil {
			return nil
		}
		switch ascii {
		case 'q', byte(3), byte(27):
			return nil
		case 'n', 'f', ' ':
			p.Render(p.pagination.CurrentPage+1, true)
			continue
		case 'p', 'b':
			p.Render(p.pagination.CurrentPage-1, true)
			continue
		case '0':
			if len(p.buff) == 0 {
				continue
			}
			fallthrough
		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			p.buff = append(p.buff, ascii)
			p.FootBarRender("")
			continue
		case byte(8), byte(127): // backspace
			if len(p.buff) > 0 {
				p.buff = p.buff[:len(p.buff)-1]
				p.FootBarRender("")
			}
			continue
		case byte(13): // backspace
			p.Render(util.Atoi(string(p.buff)), true)
			continue
			/*
				default:
					p.FootBarRender("ascii %d", ascii)
			*/

		}

		switch keyCode {
		case term.TERM_CODE_DOWN, term.TERM_CODE_RIGHT:
			p.Render(p.pagination.CurrentPage+1, true)
			continue
		case term.TERM_CODE_UP, term.TERM_CODE_LEFT:
			p.Render(p.pagination.CurrentPage-1, true)
			continue
		}

	}
}

// TermPager Pagination display when the number of results is greater than the limit
// the input struct must has
// Offset int
// Limit  int
// Pager  bool
func TermPaging(pageSize int, disablePage bool, out io.Writer, uri string, input, output interface{}, handle func(string, string, interface{}, interface{}, ...func(interface{})) error, cb ...func(interface{})) error {
	var (
		ok   bool
		err  error
		rv   reflect.Value
		resp = rest.RespTotal{}
	)
	p := &TermPager{
		out:    out,
		uri:    uri,
		input:  input,
		output: output,
		handle: handle,
		cb:     cb,
	}

	rv = reflect.Indirect(reflect.ValueOf(input))
	if rv.Kind() != reflect.Struct {
		return errors.New("expected a pointer to a struct")
	}

	if p.pagination, ok = rv.FieldByName("Pagination").Addr().Interface().(*rest.Pagination); !ok {
		return errors.New("expected Pagination field with input struct")
	}

	if p.pagination.CurrentPage == 0 {
		p.pagination.CurrentPage = 1
	}

	if pageSize == 0 {
		return errors.New("pageSize must > 0")
	} else {
		p.pagination.PageSize = pageSize
	}

	// get total
	if err := handle("GET", uri+"/cnt", input, &resp); err != nil {
		return err
	}
	p.total = int(resp.Total)

	if p.total == 0 {
		p.out.Write([]byte("No Data\n"))
		return nil
	}

	if p.total <= p.pagination.PageSize {
		goto once
	}

	if p.pagination.Dump {
		return p.Run()
	}

	if !term.IsTerminal(os.Stdout) || disablePage {
		goto once
	}

	return p.Run()

once:
	if err = p.handle("GET", p.uri, p.input, p.output, p.cb...); err != nil {
		if apierrors.IsNotFound(err) {
			p.out.Write([]byte("No Data\n"))
			return nil
		}
		return err
	}
	p.out.Write(Table(p.output))
	return nil
}
