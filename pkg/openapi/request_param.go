package openapi

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

type requestParam struct {
	header http.Header
	query  map[string][]string
	path   map[string]string
}

func (p *requestParam) setFieldValue(f *field, dstValue reflect.Value) error {
	var data []string
	var value string
	var ok bool

	key := f.name
	if key == "" {
		key = f.key
	}

	switch f.paramType {
	case PathType:
		if value, ok = p.path[key]; !ok {
			if f.required {
				return fmt.Errorf("%s must be set", key)
			}
			return nil
		}
		data = []string{value}
	case HeaderType:
		if value = p.header.Get(key); value == "" {
			if f.required {
				return fmt.Errorf("%s must be set", key)
			}
			return nil
		}
		data = []string{value}
	case QueryType:
		if data, ok = p.query[key]; !ok {
			if f.required {
				return fmt.Errorf("%s must be set", key)
			}
			return nil
		}
	default:
		panicType(f.typ, "invalid opt type")
	}

	if err := util.SetValue(dstValue, data); err != nil {
		return err
	}

	return nil
}

// struct -> request's path, query, header, data
func (p *requestParam) getFromFields(rv reflect.Value, fields structFields) error {
	klog.V(5).Info("entering openapi.scan()")

	for i, f := range fields.list {
		klog.V(11).Infof("%s[%d] %s", rv.Type(), i, f)
		subv, err := getSubv(rv, f.index, false)
		if err != nil || subv.IsZero() {
			if f.required {
				return fmt.Errorf("%v must be set", f.key)
			}
			continue
		}
		if err := p.getFromField(subv, &f); err != nil {
			klog.V(11).Infof("f %v subv %v", f, subv)
			return err
		}
	}

	return nil
}

func (p *requestParam) getFromField(srcValue reflect.Value, f *field) error {
	data, err := util.GetValue(srcValue)
	if err != nil {
		return err
	}

	key := f.name
	if key == "" {
		key = f.key
	}

	switch f.paramType {
	case PathType:
		p.path[key] = data[0]
	case QueryType:
		p.query[key] = data
	case HeaderType:
		p.header.Set(key, data[0])
	default:
		return fmt.Errorf("invalid kind: %s %s", f.paramType, f.key)
	}
	return nil

}

func invokePathVariable(rawurl string, data map[string]string) (string, error) {
	var buf strings.Builder
	var begin int

	match := false
	for i, c := range []byte(rawurl) {
		if !match {
			if c == '{' {
				match = true
				begin = i
			} else {
				buf.WriteByte(c)
			}
			continue
		}

		if c == '}' {
			k := rawurl[begin+1 : i]
			if v, ok := data[k]; ok {
				buf.WriteString(url.PathEscape(v))
			} else {
				return "", fmt.Errorf("param {%s} not found in data (%s)", k, util.JsonStr(data, true))
			}
			match = false
		}
	}

	if match {
		return "", fmt.Errorf("param %s is not ended", rawurl[begin:])
	}

	return buf.String(), nil
}
