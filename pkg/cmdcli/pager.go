package cmdcli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"

	"github.com/buger/goterm"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/term"
)

type Pager struct {
	r *Request

	// term Pager
	*rest.Pagination
	disablePage bool
	total       int
	pageTotal   int

	//render
	buff   []byte
	stdout io.Writer
}

// pageSize == 0 : no limit
func NewPager(r *Request, stdout io.Writer, disablePage bool) (*Pager, error) {
	p := &Pager{
		r:           r,
		disablePage: disablePage,
		stdout:      stdout,
	}

	if pagination, err := getPaginationFrom(r.param); err != nil {
		return nil, err
	} else {
		p.Pagination = pagination
	}

	if err := outputValidete(r.output); err != nil {
		return nil, err
	}

	return p, nil
}

func getPaginationFrom(input interface{}) (*rest.Pagination, error) {
	rv := reflect.Indirect(reflect.ValueOf(input))
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("request.input must be a pointer to a struct, got %s", rv.Kind())
	}

	pagination, ok := rv.FieldByName("Pagination").Addr().Interface().(*rest.Pagination)
	if !ok {
		return nil, errors.New("expected Pagination field with input struct")
	}

	return pagination, nil
}

func getListFrom(output interface{}) (int, interface{}) {
	rv := reflect.Indirect(reflect.ValueOf(output))

	return rv.FieldByName("Total").Interface().(int),
		rv.FieldByName("List").Interface()
}

func outputValidete(output interface{}) error {
	rv := reflect.Indirect(reflect.ValueOf(output))
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("request.output must be a pointer to a struct")
	}

	if _, ok := rv.FieldByName("Total").Interface().(int); !ok {
		return fmt.Errorf("request.output.Total must be an integer %s", rv.FieldByName("Total").Type().Kind())
	}

	v := reflect.Indirect(reflect.ValueOf(rv.FieldByName("List").Interface()))
	if !(v.Kind() == reflect.Slice || v.Kind() == reflect.Array) {
		return fmt.Errorf("request.output.List must be a slice or arrray")
	}

	return nil
}

func (p *Pager) FootBarRender(format string, a ...interface{}) {
	extra := fmt.Sprintf(format, a...)

	fmt.Fprintf(p.stdout, "\r%s %s\033[K",
		goterm.Color(fmt.Sprintf("%s/%d", string(p.buff), p.pageTotal),
			goterm.GREEN), extra)
}

func (p *Pager) Render(ctx context.Context, page int, rerend bool) (err error) {
	defer func() {
		if err == nil {
			p.buff = []byte(fmt.Sprintf("%d", page))
			p.FootBarRender("")
		}
	}()

	r := p.r

	// send query
	if page <= 0 {
		page = 1
	}
	if page >= p.pageTotal {
		page = p.pageTotal
	}
	p.CurrentPage = page
	if err = r.Do(ctx); err != nil {
		return
	}

	if rerend {
		fmt.Fprintf(p.stdout, "\033[%dA\r", p.PageSize+1)
	}

	total, list := getListFrom(r.output)
	p.total = total
	p.pageTotal = int(math.Ceil(float64(total) / float64(p.PageSize)))

	fmt.Fprintf(p.stdout, strings.Replace(string(Table(list)), "\n", "\033[K\n", -1))

	v := reflect.ValueOf(list)
	if n := p.PageSize - v.Len(); n > 0 {
		fmt.Fprintf(p.stdout, strings.Repeat("\033[K\n", n))
	}

	return
}

func (p *Pager) Dump(ctx context.Context) (err error) {
	pageTotal := 1
	p.PageSize = 100
	r := p.r

	for i := 0; i < pageTotal; i++ {
		p.CurrentPage = i
		if err = p.r.Do(ctx); err != nil {
			return
		}
		total, list := getListFrom(r.output)
		pageTotal = int(math.Ceil(float64(total) / float64(p.PageSize)))

		output := Table(list)
		if i > 0 {
			if i := bytes.IndexByte(output, '\n'); i > 0 {
				output = output[i+1:]
			}
		}
		p.stdout.Write(output)
	}
	return nil
}

func (p *Pager) Do(ctx context.Context) error {
	if p.PageSize == 0 {
		return p.Dump(ctx)
	}

	defer func() {
		// Show cursor.
		fmt.Fprintf(p.stdout, "\033[?25h\n")
	}()

	if err := p.Render(ctx, p.CurrentPage, false); err != nil {
		return err
	}

	// Hide cursor.
	fmt.Fprintf(p.stdout, "\033[?25l")

	for {
		ascii, keyCode, err := term.Getch()
		if err != nil {
			return nil
		}
		switch ascii {
		case 'q', byte(3), byte(27):
			return nil
		case 'n', 'f', ' ':
			p.Render(ctx, p.CurrentPage+1, true)
			continue
		case 'p', 'b':
			p.Render(ctx, p.CurrentPage-1, true)
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
			p.Render(ctx, util.Atoi(string(p.buff)), true)
			continue
		}

		switch keyCode {
		case term.TERM_CODE_DOWN, term.TERM_CODE_RIGHT:
			p.Render(ctx, p.CurrentPage+1, true)
			continue
		case term.TERM_CODE_UP, term.TERM_CODE_LEFT:
			p.Render(ctx, p.CurrentPage-1, true)
			continue
		}
	}
}
