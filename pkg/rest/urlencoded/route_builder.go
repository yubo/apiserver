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
	parameter := restful.FormParameter(f.key, f.description)

	switch f.typ.Kind() {
	case reflect.String:
		parameter.DataType("string")
	case reflect.Bool:
		parameter.DataType("bool")
	case reflect.Uint, reflect.Int, reflect.Int32, reflect.Int64:
		parameter.DataType("integer")
	case reflect.Slice:
		if typeName := f.typ.Elem().Name(); typeName != "string" {
			panicType(f.typ, "unsupported param")
		}
	default:
		panicType(f.typ, "unsupported type")
	}

	if f.format != "" {
		parameter.DataFormat(f.format)
	}

	b.Param(parameter)

	return nil

}
