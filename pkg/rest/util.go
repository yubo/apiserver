package rest

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/go-openapi/spec"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

type Validator interface {
	Validate() error
}

func Req2curl(req *http.Request, body []byte, inputFile, outputFile *string) string {
	buf := bytes.Buffer{}
	buf.WriteString("curl -X " + escapeShell(req.Method))

	if inputFile != nil {
		buf.WriteString(" -T " + escapeShell(*inputFile))
	}

	if outputFile != nil {
		buf.WriteString(" -o " + escapeShell(*outputFile))
	}

	if len(body) > 0 {
		data := printStr(util.SubStr3(string(body), 512, -512))
		buf.WriteString(" -d " + escapeShell(data))
	}

	var keys []string
	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		buf.WriteString(" -H " + escapeShell(fmt.Sprintf("%s: %s", k, strings.Join(req.Header[k], " "))))
	}

	buf.WriteString(" " + escapeShell(req.URL.String()))

	return buf.String()
}

func escapeShell(in string) string {
	return `'` + strings.Replace(in, `'`, `'\''`, -1) + `'`
}

// TODO: remove
func IsEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

// isVowel returns true if the rune is a vowel (case insensitive).
func isVowel(c rune) bool {
	vowels := []rune{'a', 'e', 'i', 'o', 'u'}
	for _, value := range vowels {
		if value == unicode.ToLower(c) {
			return true
		}
	}
	return false
}

func rvInfo(rv reflect.Value) {
	if klog.V(5).Enabled() {
		klog.InfoDepth(1, fmt.Sprintf("isValid %v", rv.IsValid()))
		klog.InfoDepth(1, fmt.Sprintf("rv string %s kind %s", rv.String(), rv.Kind()))
	}
}

func printStr(in string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return '.'
	}, in)
}

// github.com/emicklei/go-restful-openapi/definition_builder.go
func KeyFrom(st reflect.Type, modelTypeNameFn func(t reflect.Type) (string, bool)) string {
	key := st.String()
	if modelTypeNameFn != nil {
		if name, ok := modelTypeNameFn(st); ok {
			key = name
		}
	}
	if len(st.Name()) == 0 { // unnamed type
		// If it is an array, remove the leading []
		key = strings.TrimPrefix(key, "[]")
		// Swagger UI has special meaning for [
		key = strings.Replace(key, "[]", "||", -1)
	}
	return key
}

func OperationFrom(s *spec.Swagger, method, path string) (*spec.Operation, error) {
	p, err := s.Paths.JSONLookup(strings.TrimRight(path, "/"))
	if err != nil {
		return nil, err
	}

	var ret *spec.Operation
	pathItem := p.(*spec.PathItem)

	switch strings.ToLower(method) {
	case "get":
		ret = pathItem.Get
	case "post":
		ret = pathItem.Post
	case "patch":
		ret = pathItem.Patch
	case "delete":
		ret = pathItem.Delete
	case "put":
		ret = pathItem.Put
	case "head":
		ret = pathItem.Head
	case "options":
		ret = pathItem.Options
	default:
		// unsupported method
		return nil, fmt.Errorf("skipping Security openapi spec for %s:%s, unsupported method '%s'", method, path, method)
	}

	if ret == nil {
		return nil, fmt.Errorf("can't found %s:%s", method, path)
	}

	return ret, nil
}
