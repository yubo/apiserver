package urlencoded

import (
	"reflect"

	restful "github.com/emicklei/go-restful/v3"
)

func RouteBuilderReads(b *restful.RouteBuilder, v reflect.Value) error {
	rv := reflect.Indirect(v)
	rt := v.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		panicType(rt, "schema: interface must be a struct")
	}

	fields := cachedTypeFields(rt)

	for _, f := range fields.list {
		if err := buildParam(b, &f); err != nil {
			panic(err)
		}
	}

	return nil
}

func buildParam(b *restful.RouteBuilder, f *field) error {
	var parameter *restful.Parameter

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
