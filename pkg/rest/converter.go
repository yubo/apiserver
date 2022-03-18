package rest

import (
	"fmt"
	"reflect"

	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

const (
	PathType   = "path"
	QueryType  = "query"
	HeaderType = "header"
)

type Validator interface {
	Validate() error
}

func decodeParameters(parameters *request.Parameters, into interface{}) error {
	if parameters == nil {
		return nil
	}

	rv := reflect.ValueOf(into)
	rt := rv.Type()

	if rv.Kind() != reflect.Ptr {
		return errors.NewInternalError(fmt.Errorf("needs a pointer, got %s %s",
			rt.Kind().String(), rv.Kind().String()))
	}

	if rv.IsNil() {
		return fmt.Errorf("invalid pointer(nil)")
	}

	rv = rv.Elem()
	rt = rv.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		return fmt.Errorf("schema: interface must be a pointer to struct")
	}

	fields := cachedTypeFields(rt)
	for _, f := range fields.list {
		subv, err := getSubv(rv, f.index, true)
		if err != nil {
			return err
		}
		if err := setFieldValue(parameters, &f, subv); err != nil {
			return err
		}
	}

	return nil
}

func setFieldValue(p *request.Parameters, f *field, dstValue reflect.Value) error {
	var data []string
	var value string
	var ok bool

	key := f.name
	if key == "" {
		key = f.key
	}

	switch f.paramType {
	case PathType:
		if value, ok = p.Path[key]; !ok {
			if f.required {
				return fmt.Errorf("%s must be set", key)
			}
			return nil
		}
		data = []string{value}
	case HeaderType:
		if value = p.Header.Get(key); value == "" {
			if f.required {
				return fmt.Errorf("%s must be set", key)
			}
			return nil
		}
		data = []string{value}
	case QueryType:
		if data, ok = p.Query[key]; !ok {
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

func encodeParameters(obj interface{}) (*request.Parameters, error) {
	if v, ok := obj.(Validator); ok {
		if err := v.Validate(); err != nil {
			return nil, err
		}
	}

	rv := reflect.Indirect(reflect.ValueOf(obj))
	rt := rv.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		return nil, fmt.Errorf("rest-encode: input must be a struct, got %v/%v", rv.Kind(), rt)
	}

	params := request.NewParameters()
	fields := cachedTypeFields(rt)

	for i, f := range fields.list {
		klog.V(11).InfoS("fileds info", "index", i, "type", rv.Type(),
			"name", f.name, "key", f.key, "paramType", f.paramType,
			"skip", f.skip, "required", f.required, "hidden", f.hidden,
			"format", f.format)
		subv, err := getSubv(rv, f.index, false)
		if err != nil || subv.IsZero() {
			if f.required {
				return nil, fmt.Errorf("%v must be set", f.key)
			}
			continue
		}

		key := f.Key()
		data, err := util.GetValue(subv)
		if err != nil {
			return nil, err
		}

		switch f.paramType {
		case PathType:
			params.Path[key] = data[0]
		case QueryType:
			for i := range data {
				params.Query.Add(key, data[i])
			}
		case HeaderType:
			for i := range data {
				params.Header.Add(key, data[i])
			}
		default:
			return nil, fmt.Errorf("invalid kind: %s %s", f.paramType, key)
		}
	}

	return params, nil
}

func buildParameters(rb *restful.RouteBuilder, obj interface{}) {
	rv := reflect.Indirect(reflect.ValueOf(obj))
	rt := rv.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		panic(fmt.Sprintf("schema: interface must be a struct get %s %s", rt.Kind(), rt.String()))
	}

	fields := cachedTypeFields(rt)
	for _, f := range fields.list {
		if err := setRouteBuilderParam(rb, &f); err != nil {
			panic(err)
		}
	}
}

func setRouteBuilderParam(rb *restful.RouteBuilder, f *field) error {
	var parameter *restful.Parameter

	switch f.paramType {
	case PathType:
		parameter = restful.PathParameter(f.key, f.description)
	case QueryType:
		if f.hidden {
			return nil
		}
		parameter = restful.QueryParameter(f.key, f.description)
		parameter.Required(f.required)
	case HeaderType:
		if f.hidden {
			return nil
		}
		parameter = restful.HeaderParameter(f.key, f.description)
		parameter.Required(f.required)
	default:
		panicType(f.typ, "setParam")
	}

	switch f.typ.Kind() {
	case reflect.String:
		parameter.DataType("string")
	case reflect.Bool:
		parameter.DataType("boolean")
	case reflect.Uint, reflect.Int, reflect.Int32, reflect.Int64:
		parameter.DataType("integer")
	case reflect.Slice:
		if typeName := f.typ.Elem().Name(); typeName != "string" {
			panicType(f.typ, "unsupported param")
		}
	default:
		panicType(f.typ, "unsupported param")
	}

	if f.format != "" {
		parameter.DataFormat(f.format)
	}

	rb.Param(parameter)

	return nil
}

// parameterCodec implements conversion to and from query parameters and objects.
type ParameterCodec struct{}

//
//// DecodeParameters converts the provided url.Values into an object of type From with the kind of into, and then
//// converts that object to into (if necessary). Returns an error if the operation cannot be completed.
func (c *ParameterCodec) DecodeParameters(parameters *request.Parameters, into interface{}) error {
	return decodeParameters(parameters, into)
}

// EncodeParameters converts the provided object into the to version, then converts that object to url.Values.
// Returns an error if conversion is not possible.
func (c *ParameterCodec) EncodeParameters(obj interface{}) (*request.Parameters, error) {
	return encodeParameters(obj)
}

func (c *ParameterCodec) RouteBuilderParameters(rb *restful.RouteBuilder, obj interface{}) {
	buildParameters(rb, obj)
}

// NewParameterCodec creates a ParameterCodec capable of transforming url values into versioned objects and back.
func NewParameterCodec() request.ParameterCodec {
	return &ParameterCodec{}
}
