package cmdcli

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/yubo/golib/configer"
)

const (
	maxDepth = 5
)

// struct -> args
type CmdArg interface {
	CmdArg(string) []string
}

func TrimArgs(in []string) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = strings.TrimSpace(v)
	}
	return out
}

// `flag:""`
// `flag:"values"`
// `flag:"values,f"`

// struct -> []string
// GetArgs decode args from sample
func GetArgs(args, args2 *[]string, sample interface{}) error {
	err := getArgs(args, args2, sample, 0)
	if err != nil {
		return err
	}

	*args = TrimArgs(*args)
	return nil
}

func getArgs(args, args2 *[]string, sample interface{}, depth int) error {
	if depth > maxDepth {
		panic(fmt.Sprintf("depth is larger than the maximum allowed depth of %d", maxDepth))
	}

	rv := reflect.Indirect(reflect.ValueOf(sample))
	rt := rv.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		return errors.New("sample input must be a struct")
	}

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		fv := rv.Field(i)

		if !fv.CanInterface() {
			continue
		}

		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}

		tag := configer.GetFieldTag(sf)
		if tag.Skip() {
			continue
		}

		//
		if tag.Arg == "arg1" {
			switch v := fv.Interface().(type) {
			case []string:
				*args = append(*args, v...)
			case string:
				*args = append(*args, v)
			case CmdArg:
				*args = append(*args, v.CmdArg("")...)
			default:
				if s := fmt.Sprintf("%v", v); len(s) > 0 {
					*args = append(*args, s)
				}
			}
			//if v, ok := in.([]string); ok {
			//	*args = append(*args, v...)
			//} else if v, ok := in.(string); ok {
			//	*args = append(*args, v)
			//} else if v, ok := in.(CmdArg); ok {
			//	*args = append(*args, v.CmdArg("")...)
			//} else if v := fmt.Sprintf("%v", in); len(v) > 0 {
			//	*args = append(*args, v)
			//}
			continue
		}

		if tag.Arg == "arg2" {
			switch v := fv.Interface().(type) {
			case []string:
				*args2 = append(*args2, v...)
			case string:
				*args2 = append(*args2, v)
			case CmdArg:
				*args2 = append(*args2, v.CmdArg("")...)
			default:
				if s := fmt.Sprintf("%v", v); len(s) > 0 {
					*args2 = append(*args2, s)
				}
			}
			//in := fv.Interface()
			//if v, ok := in.([]string); ok {
			//	*args2 = append(*args2, v...)
			//} else if v, ok := in.(string); ok {
			//	*args2 = append(*args2, v)
			//} else if v, ok := in.(CmdArg); ok {
			//	*args2 = append(*args2, v.CmdArg("")...)
			//} else if v := fmt.Sprintf("%v", in); len(v) > 0 {
			//	*args2 = append(*args2, v)
			//}
			continue
		}

		// inline
		if sf.Type.Kind() == reflect.Struct {
			if err := getArgs(args, args2, fv.Interface(), depth+1); err != nil {
				return err
			}
			continue
		}

		if err := _getArgs(args, "--"+tag.Flag[0], fv); err != nil {
			return fmt.Errorf("%s.%s %s", rt.Name(), sf.Name, err.Error())
		}
	}

	return nil
}

func _getArgs(args *[]string, key string, rv reflect.Value) (err error) {
	if arg, ok := rv.Interface().(CmdArg); ok {
		*args = append(*args, arg.CmdArg(key)...)
		return
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64:
		*args = append(*args, key, strconv.FormatInt(rv.Int(), 10))
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		*args = append(*args, key, strconv.FormatUint(rv.Uint(), 10))
	case reflect.String:
		*args = append(*args, key, rv.String())
	case reflect.Bool:
		if rv.Bool() {
			*args = append(*args, key)
		} else {
			*args = append(*args, key+"=false")
		}
	case reflect.Slice:
		for i := 0; i < rv.Len(); i++ {
			fv := reflect.Indirect(rv.Index(i))
			if err = _getArgs(args, key, fv); err != nil {
				return
			}
		}
	default:
		err = fmt.Errorf("unsupported kind %s", rv.Kind().String())
	}

	return
}

// struct -> cmd flags
// CleanupArgs set ptr to nil which flags has not been changed
func CleanupArgs(fs *pflag.FlagSet, out interface{}) {
	// dlog.Debugf("CleanupArgs entering")
	rv := reflect.ValueOf(out)
	rt := rv.Type()

	if rv.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("needs a pointer, got %s %s",
			rt.Kind().String(), rv.Kind().String()))
	}

	if rv.IsNil() {
		panic("invalid pointer(nil)")
	}

	rv = rv.Elem()
	rt = rv.Type()

	cleanupArgs(fs, rv, rt, 0)
}

// rv is elem()
func cleanupArgs(fs *pflag.FlagSet, rv reflect.Value, rt reflect.Type, depth int) {
	if depth > maxDepth {
		panic(fmt.Sprintf("depth is larger than the maximum allowed depth of %d", maxDepth))
	}

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		panic("schema: interface must be a pointer to struct")
	}

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		fv := rv.Field(i)
		ft := sf.Type

		tag := configer.GetFieldTag(sf)

		if tag.Skip() {
			continue
		}

		if sf.Type.Kind() == reflect.Struct {
			if fv.Kind() == reflect.Ptr {
				fv = fv.Elem()
				ft = fv.Type()
			}
			cleanupArgs(fs, fv, ft, depth+1)
			continue
		}

		if tag.Default == "" && !fs.Changed(tag.Flag[0]) &&
			(fv.Kind() == reflect.Ptr ||
				fv.Kind() == reflect.Map ||
				fv.Kind() == reflect.Slice) {
			fv.Set(reflect.Zero(ft))
		}
	}
}
