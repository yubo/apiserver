package urlencoded

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"reflect"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/runtime/serializer"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

const (
	maxFormSize           = int64(1<<63 - 1)
	ContentTypeUrlencoded = "application/x-www-form-urlencoded"
)

func WithSerializer(options *serializer.CodecFactoryOptions) {
	options.Serializers = append(options.Serializers, serializer.SerializerType{
		AcceptContentTypes: []string{ContentTypeUrlencoded},
		ContentType:        ContentTypeUrlencoded,
		FileExtensions:     []string{},
		Serializer:         NewSerializer(),
	})
}

func NewSerializer() *Serializer {
	return &Serializer{
		identifier: runtime.Identifier("url_encoded"),
	}
}

// Serializer handles encoding versioned objects into the proper wire form
type Serializer struct {
	identifier runtime.Identifier
}

var _ runtime.Serializer = &Serializer{}
var _ restful.EntityReaderWriter = &Serializer{}

func (s *Serializer) Decode(data []byte, into runtime.Object) (runtime.Object, error) {
	err := NewDecoder(bytes.NewReader(data)).Decode(into)
	if err != nil {
		return nil, err
	}

	return into, nil
}

// Encode serializes the provided object to the given writer.
func (s *Serializer) Encode(obj runtime.Object, w io.Writer) error {
	return NewEncoder(w).Encode(obj)
}

// Identifier implements runtime.Encoder interface.
func (s *Serializer) Identifier() runtime.Identifier {
	return s.identifier
}

// Read unmarshalls the value from byte slice and using urlencoded to unmarshal
func (s *Serializer) Read(req *restful.Request, v interface{}) error {
	return NewDecoder(req.Request.Body).Form(req.Request.Form).Decode(v)
}

// Write marshals the value to byte slice and set the Content-Type Header.
func (s *Serializer) Write(resp *restful.Response, status int, v interface{}) error {
	resp.WriteHeader(status)

	if v == nil {
		// do not write a nil representation
		return nil
	}
	return NewEncoder(resp).Encode(v)
}

// encode {{{
type Encoder struct {
	writer io.Writer
	values url.Values
}

func NewEncoder(w io.Writer) *Encoder {
	klog.V(5).Info("encoder entering")
	return &Encoder{writer: w, values: make(url.Values)}
}

func (p *Encoder) Encode(src interface{}) error {
	// struct -> values
	if err := p.scan(src); err != nil {
		return err
	}

	if _, err := p.writer.Write([]byte(p.values.Encode())); err != nil {
		return err
	}

	klog.V(5).Infof("encoded context %s", p.values.Encode())
	return nil
}

// scanMap not support inline model yet
func (p *Encoder) scanMap(src map[string]interface{}) error {
	// klog.V(5).Infof("scanMap entering %s", util.JsonStr(src, true))
	for k, v := range src {
		rv := reflect.Indirect(reflect.ValueOf(v))
		rt := rv.Type()

		if rv.Kind() == reflect.Struct &&
			rt.String() != "time.Time" {
			p.scan(v)
			continue
		}
		data, err := util.GetValue(rv)
		if err != nil {
			return err
		}

		if len(data) > 0 {
			p.values[k] = data
		}
	}
	return nil
}

// struct -> values
func (p *Encoder) scan(src interface{}) error {
	// map[string]interface{}
	if v, ok := src.(map[string]interface{}); ok {
		return p.scanMap(v)
	}

	rv := reflect.Indirect(reflect.ValueOf(src))
	rt := rv.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		return fmt.Errorf("schema: interface must be a struct got %s", rt.String())
	}

	fields := cachedTypeFields(rt)
	for _, f := range fields.list {
		subv, err := getSubv(rv, f.index, false)
		if err != nil || subv.IsZero() {
			if f.Required {
				return fmt.Errorf("%v must be set", f.Key)
			}
			continue
		}

		key := f.Key
		data, err := util.GetValue(subv)
		if err != nil {
			return err
		}

		for i := range data {
			p.values.Add(key, data[i])
		}
	}

	return nil
}

// }}}

// decode {{{
type Decoder struct {
	reader io.Reader
	values url.Values
	form   url.Values
}

func NewDecoder(r io.Reader) *Decoder {
	klog.V(5).Infof("decoder entering")
	return &Decoder{reader: r}
}

func (p *Decoder) Form(form url.Values) *Decoder {
	p.form = form
	return p
}

func (p *Decoder) Decode(dst interface{}) error {
	if p.form != nil {
		p.values = p.form
	} else {
		b, err := ioutil.ReadAll(p.reader)
		if err != nil {
			return err
		}

		klog.V(5).Infof("decode body %s", string(b))

		if int64(len(b)) > maxFormSize {
			return errors.NewRequestEntityTooLargeError("http body")
		}

		p.values, err = url.ParseQuery(string(b))
		if err != nil {
			return err
		}
	}

	rv := reflect.ValueOf(dst)
	rt := rv.Type()

	if rv.Kind() != reflect.Ptr {
		return errors.NewInternalError(fmt.Errorf("needs a pointer, got %s %s", rt.Kind().String(), rv.Kind().String()))
	}

	if rv.IsNil() {
		return errors.NewInternalError(fmt.Errorf("invalid potiner(nil)"))
	}

	rv = rv.Elem()
	rt = rv.Type()

	return p.decode(rv, rt)
}

func (p *Decoder) decode(rv reflect.Value, rt reflect.Type) error {
	klog.V(5).Infof("entering decode")

	if rv.Kind() != reflect.Struct || rv.Kind() == reflect.Slice || rt.String() == "time.Time" {
		return errors.NewInternalError(fmt.Errorf("schema: interface must be a pointer to struct"))
	}

	fields := cachedTypeFields(rt)
	for _, f := range fields.list {
		subv, err := getSubv(rv, f.index, true)
		if err != nil {
			return err
		}

		if err := util.SetValue(subv, p.values[f.Key]); err != nil {
			return err
		}

	}

	return nil
}

// }}}
