package cmdcli

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// struct -> args {{{

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

// `flags:""`
// `flags:"-"`
// `flags:",arg"`
// `flags:"values,,"`
// `flags:"values,f,"`

// struct -> []string
// GetArgs decode args from sample
func GetArgs(args, args2 *[]string, sample interface{}) error {
	err := getArgs(args, args2, sample)
	if err != nil {
		return err
	}

	*args = TrimArgs(*args)
	return nil
}

func getArgs(args, args2 *[]string, sample interface{}) error {
	rv := reflect.Indirect(reflect.ValueOf(sample))
	rt := rv.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		return errors.New("sample input must be a struct")
	}

	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ff := rt.Field(i)

		if !fv.CanInterface() {
			continue
		}

		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}

		flags, _, skip, arg, arg2, local, inline, _, err := getTags(ff)
		if err != nil {
			panic(err)
		}

		if skip || local {
			continue
		}

		if arg {
			in := fv.Interface()
			if v, ok := in.([]string); ok {
				*args = append(*args, v...)
			} else if v, ok := in.(string); ok {
				*args = append(*args, v)
			} else if v, ok := in.(CmdArg); ok {
				*args = append(*args, v.CmdArg("")...)
			} else if v := fmt.Sprintf("%v", in); len(v) > 0 {
				*args = append(*args, v)
			}
			continue
		}

		if arg2 {
			in := fv.Interface()
			if v, ok := in.([]string); ok {
				*args2 = append(*args2, v...)
			} else if v, ok := in.(string); ok {
				*args2 = append(*args2, v)
			} else if v, ok := in.(CmdArg); ok {
				*args2 = append(*args2, v.CmdArg("")...)
			} else if v := fmt.Sprintf("%v", in); len(v) > 0 {
				*args2 = append(*args2, v)
			}
			continue
		}

		if inline {
			if err := getArgs(args, args2, fv.Interface()); err != nil {
				return err
			}
			continue
		}

		if err := getArgs2(args, "--"+flags[0], fv); err != nil {
			return fmt.Errorf("%s.%s %s", rt.Name(), ff.Name, err.Error())
		}
	}

	return nil
}

func getArgs2(args *[]string, key string, rv reflect.Value) (err error) {

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
			if err = getArgs2(args, key, fv); err != nil {
				return
			}
		}
	default:
		err = fmt.Errorf("unsupported kind %s", rv.Kind().String())
	}

	return

}

// }}}

// struct -> pflag.Addflags -> cml help/usage {{{
func AddFlags(fs *pflag.FlagSet, in interface{}) {
	if reflect.ValueOf(in).Kind() != reflect.Ptr {
		panic("must ptr")
	}

	rv := reflect.ValueOf(in).Elem()
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ff := rt.Field(i)
		ft := ff.Type

		flags, def, skip, arg, arg2, _, inline, desc, err := getTags(ff)
		if err != nil {
			panic(err)
		}

		if skip || arg || arg2 {
			continue
		}

		if inline {
			prepareValue(fv, ft)
			if fv.Kind() == reflect.Ptr {
				fv = fv.Elem()
				ft = fv.Type()
			}
			AddFlags(fs, fv.Addr().Interface())
			continue
		}

		addFlag(fs, fv, ft, flags, def, desc)
	}
}

func addFlag(fs *pflag.FlagSet, rv reflect.Value, rt reflect.Type, flags []string, def, desc string) {
	if !rv.CanSet() {
		panic(fmt.Sprintf("%v(%s) can not be set", flags, rv.Kind()))
	}

	if len(flags) != 2 {
		panic(fmt.Sprintf("%v(%s) len != 2", flags, rv.Kind()))
	}

	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rt.Elem()))
		}
		rv = rv.Elem()
	}

	p := rv.Addr().Interface()

	if v, ok := p.(pflag.Value); ok {
		if len(flags) == 1 {
			fs.Var(v, flags[0], desc)
		} else if len(flags) == 2 {
			fs.VarP(v, flags[0], flags[1], desc)
		}
		return
	}

	if flags[1] == "" {
		switch rv.Kind() {
		case reflect.String:
			fs.StringVar(p.(*string), flags[0], def, desc)
		case reflect.Bool:
			fs.BoolVar(p.(*bool), flags[0], toBool(def), desc)
		case reflect.Uint:
			fs.UintVar(p.(*uint), flags[0], uint(toInt64(def)), desc)
		case reflect.Int:
			fs.IntVar(p.(*int), flags[0], int(toInt64(def)), desc)
		case reflect.Int32:
			fs.Int32Var(p.(*int32), flags[0], int32(toInt64(def)), desc)
		case reflect.Int64:
			fs.Int64Var(p.(*int64), flags[0], toInt64(def), desc)
		case reflect.Slice:
			typeName := rt.Elem().String()
			if typeName == "string" {
				fs.StringArrayVar(p.(*[]string), flags[0], strings.Split(def, ","), desc)
			} else {
				panic(fmt.Sprintf("unsupported type flags %s slice", typeName))
			}
		default:
			panic("not support flags type " + rt.String())
		}
		return
	}

	// len(flags) == 2
	switch rv.Kind() {
	case reflect.String:
		fs.StringVarP(p.(*string), flags[0], flags[1], def, desc)
	case reflect.Bool:
		fs.BoolVarP(p.(*bool), flags[0], flags[1], toBool(def), desc)
	case reflect.Uint:
		fs.UintVarP(p.(*uint), flags[0], flags[1], uint(toInt64(def)), desc)
	case reflect.Int:
		fs.IntVarP(p.(*int), flags[0], flags[1], int(toInt64(def)), desc)
	case reflect.Int32:
		fs.Int32VarP(p.(*int32), flags[0], flags[1], int32(toInt64(def)), desc)
	case reflect.Int64:
		fs.Int64VarP(p.(*int64), flags[0], flags[1], toInt64(def), desc)
	case reflect.Slice:
		typeName := rt.Elem().String()
		if typeName == "string" {
			fs.StringArrayVarP(p.(*[]string), flags[0], flags[1], strings.Split(def, ","), desc)
		} else {
			panic(fmt.Sprintf("unsupported type flags %s slice", typeName))
		}
	default:
		panic("not support flags type " + rt.String())
	}
}

// }}}

// struct -> cmd flags {{{
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

	cleanupArgs(fs, rv, rt)
}

// rv is elem()
func cleanupArgs(fs *pflag.FlagSet, rv reflect.Value, rt reflect.Type) {
	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		panic("schema: interface must be a pointer to struct")
	}

	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ff := rt.Field(i)
		ft := ff.Type

		flags, def, skip, arg, arg2, _, inline, _, err := getTags(ff)
		if err != nil {
			panic(err)
		}

		if skip || arg || arg2 {
			// dlog.Debugf("skip %v arg %v", skip, arg)
			continue
		}

		if inline {
			if fv.Kind() == reflect.Ptr {
				fv = fv.Elem()
				ft = fv.Type()
			}
			cleanupArgs(fs, fv, ft)
			continue
		}

		if def == "" && !fs.Changed(flags[0]) &&
			(fv.Kind() == reflect.Ptr ||
				fv.Kind() == reflect.Map ||
				fv.Kind() == reflect.Slice) {
			fv.Set(reflect.Zero(ft))
		}
	}
}

// }}}
func toBool(in string) bool {
	if in == "true" {
		return true
	}
	return false
}

func toInt64(in string) int64 {
	iv, err := strconv.ParseInt(in, 10, 64)
	if err != nil {
		return 0
	}
	return iv
}

func CheckArgsLength(argsReceived int, requiredArgs ...string) error {
	expectedNum := len(requiredArgs)
	if argsReceived != expectedNum {
		arg := "arguments"
		if expectedNum == 1 {
			arg = "argument"
		}

		e := fmt.Sprintf("This command needs %v %s:", expectedNum, arg)
		for _, v := range requiredArgs {
			e += " <" + v + ">"
		}
		return fmt.Errorf(e)
	}
	return nil
}

func CheckArgsLength3(args []string, atLeast int, requiredArgs ...string) (string, string, string, error) {
	if len(args) < atLeast {
		arg := "arguments"
		if atLeast == 1 {
			arg = "argument"
		}

		e := fmt.Sprintf("This command needs %v %s:", atLeast, arg)
		for i, v := range requiredArgs {
			if i < atLeast {
				e += " <" + v + ">"
			} else {
				e += " [" + v + "]"
			}
		}
		return "", "", "", fmt.Errorf(e)
	}

	r := [3]string{}
	for i := 0; i < 3 && i < len(args); i++ {
		r[i] = args[i]
	}
	return r[0], r[1], r[2], nil
}

func getTags(ft reflect.StructField) (flags []string, def string, skip, arg, arg2, local, inline bool, desc string, err error) {
	tag, ok := ft.Tag.Lookup("flags")
	if !ok || tag == "-" {
		skip = true
		return
	}

	if tag == ",inline" {
		inline = true
		return
	}

	if tag == ",arg" {
		arg = true
		return
	}

	if tag == ",arg2" {
		arg2 = true
		return
	}

	if t, _ := ft.Tag.Lookup("local"); t == "true" {
		local = true
	}

	flags = strings.Split(tag, ",")

	if len(flags) < 3 {
		err = fmt.Errorf("tag(%s) is invalid, format flags:{long},{short},{default}", tag)
	}

	def = strings.Join(flags[2:], ",")
	flags = flags[:2]
	if len(flags[1]) > 1 {
		err = fmt.Errorf("tag(%s) shot option name(%s) is invalid", tag, flags[1])
	}

	desc = ft.Tag.Get("description")

	return
}

func prepareValue(rv reflect.Value, rt reflect.Type) {
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		rv.Set(reflect.New(rt.Elem()))
	}
}
