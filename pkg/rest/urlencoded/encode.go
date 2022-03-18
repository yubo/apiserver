package urlencoded

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"reflect"

	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

func Marshal(in interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := NewEncoder(&buf).Encode(in)
	return buf.Bytes(), err
}

type Encoder struct {
	w      io.Writer
	values url.Values
}

func NewEncoder(w io.Writer) *Encoder {
	klog.V(5).Info("encoder entering")
	return &Encoder{w: w, values: make(url.Values)}
}

func (p *Encoder) Encode(src interface{}) error {

	// struct -> values
	if err := p.scan(src); err != nil {
		return err
	}

	if _, err := p.w.Write([]byte(p.values.Encode())); err != nil {
		return err
	}

	klog.V(5).Infof("encoded context %s", p.values.Encode())
	return nil
}

// scanMap not support inline model yet
func (p *Encoder) scanMap(src map[string]interface{}) error {
	// klog.V(5).Infof("scanMap entering %s", util.JsonStr(src, true))
	for k, v := range src {
		rv := reflect.Indirect(reflect.ValueOf(v))
		rt := rv.Type()

		if rv.Kind() == reflect.Struct &&
			rt.String() != "time.Time" {
			p.scan(v)
			continue
		}
		data, err := util.GetValue(rv)
		if err != nil {
			return err
		}

		if len(data) > 0 {
			p.values[k] = data
		}
	}
	return nil
}

// struct -> values
func (p *Encoder) scan(src interface{}) error {
	// map[string]interface{}
	if v, ok := src.(map[string]interface{}); ok {
		return p.scanMap(v)
	}

	rv := reflect.Indirect(reflect.ValueOf(src))
	rt := rv.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		return fmt.Errorf("schema: interface must be a struct got %s", rt.String())
	}

	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ff := rt.Field(i)

		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}

		if !fv.CanInterface() {
			continue
		}

		name, _, skip, inline := getTags(ff)
		if skip {
			continue
		}

		if inline {
			if err := p.scan(fv.Interface()); err != nil {
				return err
			}
			continue
		}

		data, err := util.GetValue(fv)
		if err != nil {
			return err
		}

		if len(data) > 0 {
			p.values[name] = data
		}
	}

	return nil
}
