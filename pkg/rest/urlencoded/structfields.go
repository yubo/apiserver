package urlencoded

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/yubo/golib/util"
)

const (
	DataType = "data"
)

var fieldCache sync.Map // map[reflect.Type]structFields

// A field represents a single field found in a struct.
// `param:"query,required" format:"password" description:"aaa"`
type field struct {
	fieldProps
	Type  reflect.Type
	index []int
}

type fieldProps struct {
	Key string

	Skip     bool
	Hidden   bool // json:<type>[,hiddent]
	Required bool

	Name        string
	Format      string
	Description string
	Enum        []string // enum:<a|b|c>
	Maximum     *float64 // maximum: 500
	Minimum     *float64 // minimum: 10
	Default     string
}

type structFields struct {
	list      []field
	nameIndex map[string]int
}

// cachedTypeFields is like typeFields but uses a cache to avoid repeated work.
func cachedTypeFields(t reflect.Type) structFields {
	if f, ok := fieldCache.Load(t); ok {
		return f.(structFields)
	}
	f, _ := fieldCache.LoadOrStore(t, typeFields(t))
	return f.(structFields)
}

// typeFields returns a list of fields that JSON should recognize for the given type.
// The algorithm is breadth-first search over the set of structs to include - the top struct
// and then any reachable anonymous structs.
func typeFields(t reflect.Type) structFields {
	// Anonymous fields to explore at the current level and the next.
	current := []field{}
	next := []field{{Type: t}}

	// Count of queued names for current level and the next.
	var count, nextCount map[reflect.Type]int

	// Types already visited at an earlier level.
	visited := map[reflect.Type]bool{}

	// Fields found.
	var fields []field

	// Buffer to run HTMLEscape on field names.
	// var nameEscBuf bytes.Buffer

	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, map[reflect.Type]int{}

		for _, f := range current {
			if visited[f.Type] {
				continue
			}
			visited[f.Type] = true

			// Scan f.typ for fields to include.
			for i := 0; i < f.Type.NumField(); i++ {
				sf := f.Type.Field(i)
				isUnexported := sf.PkgPath != ""
				if sf.Anonymous {
					t := sf.Type
					if t.Kind() == reflect.Ptr {
						t = t.Elem()
					}
					if isUnexported && t.Kind() != reflect.Struct {
						// Ignore embedded fields of unexported non-struct types.
						continue
					}
					// Do not ignore embedded fields of unexported struct types
					// since they may have exported fields.
				} else if isUnexported {
					// Ignore unexported non-embedded fields.
					continue
				}
				opt := getFieldProps(sf)
				if opt.Skip {
					continue
				}
				index := make([]int, len(f.index)+1)
				copy(index, f.index)
				index[len(f.index)] = i

				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Ptr {
					// Follow pointer.
					ft = ft.Elem()
				}

				// Record found field and index sequence.
				if opt.Name != "" || !sf.Anonymous || ft.Kind() != reflect.Struct {
					field := field{
						fieldProps: opt,
						index:      index,
						Type:       ft,
					}

					fields = append(fields, field)
					if count[f.Type] > 1 {
						// If there were multiple instances, add a second,
						// so that the annihilation code will see a duplicate.
						// It only cares about the distinction between 1 or 2,
						// so don't bother generating any more copies.
						fields = append(fields, fields[len(fields)-1])
					}
					continue
				}

				// Record new anonymous struct to explore in next round.
				nextCount[ft]++
				if nextCount[ft] == 1 {
					next = append(next, field{index: index, Type: ft})
				}
			}
		}
	}

	nameIndex := make(map[string]int, len(fields))
	for i, field := range fields {
		if _, ok := nameIndex[field.Key]; ok {
			panicType(field.Type, t.Name(), field)
		}
		nameIndex[field.Key] = i
	}
	return structFields{fields, nameIndex}
}

func getSubv(rv reflect.Value, index []int, allowCreate bool) (reflect.Value, error) {
	subv := rv
	for _, i := range index {
		if subv.Kind() == reflect.Ptr {
			if subv.IsNil() {
				if !allowCreate {
					return subv, fmt.Errorf("struct %v is nil", subv.Type().Elem())
				}

				if !subv.CanSet() {
					return subv, fmt.Errorf("getSubv: cannot set embedded pointer to unexported struct: %v", subv.Type().Elem())
				}
				subv.Set(reflect.New(subv.Type().Elem()))
			}
			subv = subv.Elem()
		}
		subv = subv.Field(i)
	}
	return subv, nil
}

// `json:"[name](,required)?"`
// `format:"password"`
// `description:"ooxxoo"`
// func getTags(ff reflect.StructField) (name, paramType, format string, skip bool) {
func getFieldProps(sf reflect.StructField) (opt fieldProps) {
	if sf.Anonymous {
		return
	}

	tag := sf.Tag.Get("json")
	if tag == "-" || tag == "" {
		opt.Skip = true
		return
	}
	name, opts := parseTag(tag)
	if opts.Contains("required") {
		opt.Required = true
	}
	if opts.Contains("hidden") {
		opt.Hidden = true
	}

	opt.Name = name
	opt.Format = sf.Tag.Get("format")
	opt.Description = sf.Tag.Get("description")
	opt.Default = sf.Tag.Get("default")

	if v := sf.Tag.Get("enum"); v != "" {
		opt.Enum = strings.Split(v, "|")
	}

	if v := sf.Tag.Get("maximum"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			panic(err)
		}
		opt.Maximum = &f
	}

	if v := sf.Tag.Get("minimum"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			panic(err)
		}
		opt.Minimum = &f
	}

	if opt.Name == "" {
		opt.Key = util.LowerCamelCasedName(sf.Name)
	} else {
		opt.Key = opt.Name
	}

	return
}

// tagOptions is the string following a comma in a struct field's "json"
// tag, or the empty string. It does not include the leading comma.
type tagOptions string

// parseTag splits a struct field's json tag into its name and
// comma-separated options.
func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, tagOptions("")
}

// Contains reports whether a comma-separated list of options
// contains a particular substr flag. substr must be surrounded by a
// string boundary or commas.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}

func panicType(ft reflect.Type, args ...interface{}) {
	msg := fmt.Sprintf("type field %s %s", ft.PkgPath(), ft.Name())

	if len(args) > 0 {
		panic(fmt.Sprint(args...) + " " + msg)
	}
	panic(msg)
}
