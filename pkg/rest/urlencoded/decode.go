package urlencoded

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"reflect"
	"strings"

	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

// Unmarshal decodes the url encoded data and stores the result
// in the value pointed to by in.
func Unmarshal(data []byte, in interface{}) error {
	return NewDecoder(bytes.NewReader(data)).Decode(in)
}

type Decoder struct {
	r      io.Reader
	values url.Values
	form   url.Values
}

func NewDecoder(r io.Reader) *Decoder {
	klog.V(5).Infof("decoder entering")
	return &Decoder{r: r}
}

func (p *Decoder) Form(form url.Values) *Decoder {
	p.form = form
	return p
}

func (p *Decoder) Decode(dst interface{}) error {
	if p.form != nil {
		p.values = p.form
	} else {
		b, err := ioutil.ReadAll(p.r)
		if err != nil {
			return err
		}

		klog.V(5).Infof("decode body %s", string(b))

		if int64(len(b)) > maxFormSize {
			return errors.NewRequestEntityTooLargeError("http body")
		}

		p.values, err = url.ParseQuery(string(b))
		if err != nil {
			return err
		}
	}

	rv := reflect.ValueOf(dst)
	rt := rv.Type()

	if rv.Kind() != reflect.Ptr {
		return errors.NewInternalError(fmt.Errorf("needs a pointer, got %s %s", rt.Kind().String(), rv.Kind().String()))
	}

	if rv.IsNil() {
		return errors.NewInternalError(fmt.Errorf("invalid potiner(nil)"))
	}

	rv = rv.Elem()
	rt = rv.Type()

	return p.decode(rv, rt)
}

func (p *Decoder) decode(rv reflect.Value, rt reflect.Type) error {
	klog.V(5).Infof("entering decode")

	if rv.Kind() != reflect.Struct || rv.Kind() == reflect.Slice || rt.String() == "time.Time" {
		return errors.NewInternalError(fmt.Errorf("schema: interface must be a pointer to struct"))
	}

	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ff := rt.Field(i)
		ft := ff.Type

		name, _, skip, inline := getTags(ff)

		if !fv.CanSet() {
			klog.V(5).Infof("can't addr name %s, continue", name)
			continue
		}

		if skip {
			continue
		}

		if inline {
			// use addr() let fv can set
			util.PrepareValue(fv, ft)
			if err := p.decode(fv, ft); err != nil {
				return err
			}
			continue
		}

		if err := util.SetValue(fv, p.values[name]); err != nil {
			return err
		}

	}
	return nil
}

// `name:"name?(,inline|{format})?"`
func getTags(rf reflect.StructField) (name, format string, skip, inline bool) {
	tag, _ := rf.Tag.Lookup("name")
	if tag == "-" {
		skip = true
		return
	}

	if strings.HasSuffix(tag, ",inline") {
		inline = true
		return
	}

	tags := strings.Split(tag, ",")
	if len(tags) > 1 {
		format = tags[1]
	}

	if tags[0] != "" {
		name = tags[0]
		return
	}

	name = util.LowerCamelCasedName(rf.Name)
	return
}
