package rest

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"

	"github.com/yubo/golib/util"
)

var fieldCache sync.Map // map[reflect.Type]structFields

// A field represents a single field found in a struct.
// `param:"query,required,hidden" format:"password" description:"aaa"`
type field struct {
	tagOpt
	typ   reflect.Type
	index []int
}

func (p field) String() string {
	return fmt.Sprintf("key %s index %v %s", p.key, p.index, p.tagOpt)
}

func (p field) Key() string {
	if p.name != "" {
		return p.name
	}

	return p.key
}

type tagOpt struct {
	name        string
	key         string
	paramType   string
	format      string
	skip        bool
	hidden      bool
	required    bool
	description string
}

func (p tagOpt) String() string {
	return fmt.Sprintf("name=%s key=%v paramType=%s skip=%v required=%v hidden=%v format=%s description=%s",
		p.name, p.key, p.paramType, p.skip, p.required, p.hidden, p.format, p.description)
}

type structFields struct {
	list      []field
	nameIndex map[string]int
}

func (p structFields) String() string {
	var ret string
	for k, v := range p.list {
		ret += fmt.Sprintf("%d %s\n", k, v)
	}
	return ret
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
	next := []field{{typ: t}}

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
			if visited[f.typ] {
				continue
			}
			visited[f.typ] = true

			// Scan f.typ for fields to include.
			for i := 0; i < f.typ.NumField(); i++ {
				sf := f.typ.Field(i)
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

				opt := getTagOpt(sf)
				if opt.skip {
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
				if opt.name != "" || !sf.Anonymous || ft.Kind() != reflect.Struct {
					field := field{
						tagOpt: opt,
						index:  index,
						typ:    ft,
					}

					fields = append(fields, field)
					if count[f.typ] > 1 {
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
					next = append(next, field{index: index, typ: ft})
				}
			}
		}
	}

	nameIndex := make(map[string]int, len(fields))
	for i, field := range fields {
		if _, ok := nameIndex[field.key]; ok {
			panicType(field.typ, fmt.Sprintf("duplicate field %s", field.key))
		}
		nameIndex[field.key] = i
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

// `param:"(path|header|param|data)?(,required|hidden)?"`
// `name:"keyName"`
// `json:"keyName"`
// `format:"password"`
// `description:"ooxxoo"`
// func getTags(ff reflect.StructField) (name, paramType, format string, skip, bool) {
func getTagOpt(sf reflect.StructField) (opt tagOpt) {
	if sf.Anonymous {
		return
	}

	tag := sf.Tag.Get("param")
	if tag == "-" || tag == "" {
		opt.skip = true
		return
	}

	typ, opts := parseTag(tag)
	if opts.Contains("required") {
		opt.required = true
	}
	if opts.Contains("hidden") {
		opt.hidden = true
	}

	opt.paramType = typ
	opt.name = sf.Tag.Get("name")
	opt.format = sf.Tag.Get("format")
	opt.description = sf.Tag.Get("description")

	switch typ {
	case PathType:
		opt.key = strings.ToLower(sf.Name)
	case HeaderType:
		opt.key = strings.ToUpper(sf.Name)
	case QueryType:
		opt.key = util.LowerCamelCasedName(sf.Name)
	default:
		panicType(sf.Type, fmt.Sprintf("unknown param type=%s", typ))
	}

	if opt.name != "" {
		opt.key = opt.name
	}

	return
}

func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:<=>?@[]^_{|}~ ", c):
			// Backslash and quote chars are reserved, but
			// otherwise any punctuation chars are allowed
			// in a tag name.
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}

func panicType(ft reflect.Type, args ...interface{}) {
	msg := fmt.Sprintf("type field %s %s", ft.PkgPath(), ft.Name())

	if len(args) > 0 {
		panic(fmt.Sprint(args...) + " " + msg)
	}
	panic(msg)
}
