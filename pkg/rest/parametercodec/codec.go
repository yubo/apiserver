package parametercodec

import (
	"fmt"
	"reflect"

	"github.com/emicklei/go-restful/v3"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

const (
	PathType   = "path"
	QueryType  = "query"
	HeaderType = "header"
)

// New creates a ParameterCodec capable of transforming url values into versioned objects and back.
func New() request.ParameterCodec {
	return &parameterCodec{}
}

// parameterCodec {{{

var _ request.ParameterCodec = &parameterCodec{}

// parameterCodec implements conversion to and from query parameters and objects.
type parameterCodec struct{}

// DecodeParameters converts the provided url.Values into an object of type From with the kind of into, and then
// converts that object to into (if necessary). Returns an error if the operation cannot be completed.
func (c *parameterCodec) DecodeParameters(parameters *api.Parameters, into interface{}) error {
	return decodeParameters(parameters, into)
}

// EncodeParameters converts the provided object into the to version, then converts that object to url.Values.
// Returns an error if conversion is not possible.
func (c *parameterCodec) EncodeParameters(obj interface{}) (*api.Parameters, error) {
	return encodeParameters(obj)
}

func (c *parameterCodec) RouteBuilderParameters(b *restful.RouteBuilder, obj interface{}) {
	buildParameters(b, obj)
}

func decodeParameters(parameters *api.Parameters, into interface{}) error {
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

func (c *parameterCodec) ValidateParamType(rt reflect.Type) error {
	if rt.String() == "time.Time" {
		return fmt.Errorf("schema: interface must be a pointer to struct")
	}

	fields := cachedTypeFields(rt)
	if len(fields.list) == 0 {
		return fmt.Errorf("schema: can not find param tag from struct")
	}
	return nil
}

func setFieldValue(p *api.Parameters, f *field, dstValue reflect.Value) error {
	var data []string
	var value string
	var ok bool

	key := f.Name
	if key == "" {
		key = f.Key
	}

	switch f.ParamType {
	case PathType:
		if value, ok = p.Path[key]; !ok {
			if f.Required {
				return fmt.Errorf("%s must be set", key)
			}
			return nil
		}
		data = []string{value}
	case HeaderType:
		if value = p.Header.Get(key); value == "" {
			if f.Required {
				return fmt.Errorf("%s must be set", key)
			}
			return nil
		}
		data = []string{value}
	case QueryType:
		if data, ok = p.Query[key]; !ok {
			if f.Required {
				return fmt.Errorf("%s must be set", key)
			}
			return nil
		}
	default:
		panicType(f.Type, "invalid opt type")
	}

	if err := util.SetValue(dstValue, data); err != nil {
		return err
	}

	return nil
}

type Validator interface {
	Validate() error
}

func encodeParameters(obj interface{}) (*api.Parameters, error) {
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
		klog.V(10).InfoS("fileds info", "index", i, "type", rv.Type(),
			"name", f.Name, "key", f.Key, "paramType", f.ParamType,
			"skip", f.Skip, "required", f.Required, "hidden", f.Hidden,
			"format", f.Format)
		subv, err := getSubv(rv, f.index, false)
		if err != nil || subv.IsZero() {
			if f.Required {
				return nil, fmt.Errorf("%v must be set", f.Key)
			}
			continue
		}

		key := f.Key
		data, err := util.GetValue(subv)
		if err != nil {
			return nil, err
		}

		switch f.ParamType {
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
			return nil, fmt.Errorf("invalid kind: %s %s", f.ParamType, key)
		}
	}

	return params, nil
}

func buildParameters(b *restful.RouteBuilder, obj interface{}) {
	rv := reflect.Indirect(reflect.ValueOf(obj))
	rt := rv.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		panic(fmt.Sprintf("schema: interface must be a struct get %s %s", rt.Kind(), rt.String()))
	}

	fields := cachedTypeFields(rt)
	for _, f := range fields.list {
		if err := buildParam(b, &f); err != nil {
			panic(err)
		}
	}
}

func buildParam(b *restful.RouteBuilder, f *field) error {
	var parameter *restful.Parameter

	switch f.ParamType {
	case PathType:
		parameter = restful.PathParameter(f.Key, f.Description)
	case QueryType:
		if f.Hidden {
			return nil
		}
		parameter = restful.QueryParameter(f.Key, f.Description)
	case HeaderType:
		if f.Hidden {
			return nil
		}
		parameter = restful.HeaderParameter(f.Key, f.Description)
	default:
		panicType(f.Type, "setParam")
	}

	switch f.Type.Kind() {
	case reflect.String:
		parameter.DataType("string")
	case reflect.Bool:
		parameter.DataType("boolean")
	case reflect.Uint, reflect.Int, reflect.Int32, reflect.Int64:
		parameter.DataType("integer")
	case reflect.Slice:
		if typeName := f.Type.Elem().Name(); typeName != "string" {
			panicType(f.Type, "unsupported param")
		}
	default:
		panicType(f.Type, "unsupported param")
	}

	if f.Required {
		parameter.Required(true)
	}

	if f.Minimum != nil {
		parameter.Minimum(*f.Minimum)
	}
	if f.Maximum != nil {
		parameter.Maximum(*f.Maximum)
	}

	b.Param(parameter.
		DataFormat(f.Format).
		DefaultValue(f.Default).
		PossibleValues(f.Enum),
	)

	return nil
}

// }}}
