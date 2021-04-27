package openapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/openapi/urlencoded"
	"github.com/yubo/golib/staging/api/errors"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

type key int

const (
	// paramKey is the context key for the request param
	paramKey key = iota

	// bodyKey is the context key for the request body
	bodyKey

	// userKey is the context key for the request user.
	userKey
)

type Validator interface {
	Validate() error
}

// WithParam returns a copy of parent in which the param value is set
func WithParam(parent context.Context, param interface{}) context.Context {
	return WithValue(parent, paramKey, param)
}

// ParamFrom returns the value of the param key on the ctx
func ParamFrom(ctx context.Context) interface{} {
	return ctx.Value(paramKey)
}

// WithBody returns a copy of parent in which the body value is set
func WithBody(parent context.Context, body interface{}) context.Context {
	return WithValue(parent, bodyKey, body)
}

// BodyFrom returns the value of the param key on the ctx
func BodyFrom(ctx context.Context) interface{} {
	return ctx.Value(bodyKey)
}

// WithUser returns a copy of parent in which the user value is set
func WithUser(parent context.Context, user user.Info) context.Context {
	return WithValue(parent, userKey, user)
}

// UserFrom returns the value of the user key on the ctx
func UserFrom(ctx context.Context) (user.Info, bool) {
	user, ok := ctx.Value(userKey).(user.Info)
	return user, ok
}

// WithValue returns a copy of parent in which the value associated with key is val.
func WithValue(parent context.Context, key interface{}, val interface{}) context.Context {
	return context.WithValue(parent, key, val)
}

func HttpRequest(in *RequestOptions) (*http.Request, *http.Response, error) {
	req, err := NewRequest(in)
	if err != nil {
		return nil, nil, err
	}

	resp, err := req.Do()

	return req.Request, resp, err
}

type Request struct {
	*RequestOptions
	Request        *http.Request
	url            string
	bodyContent    []byte
	bodyReader     io.Reader
	bodyCloser     io.Closer
	responseWriter io.Writer
	responseCloser io.Closer
}

func NewRequest(in *RequestOptions, opts ...RequestOption) (req *Request, err error) {
	if in.header == nil {
		in.header = make(http.Header)
	}

	for _, opt := range opts {
		opt.apply(in)
	}

	req = &Request{RequestOptions: in}

	if err = req.prepare(); err != nil {
		return nil, err
	}

	req.Request, err = http.NewRequest(req.Method, req.url, req.bodyReader)
	if err != nil {
		return nil, err
	}

	req.Request.Header = req.header

	klog.V(10).Infof("req %s", req)
	return req, nil
}

func (p *Request) prepare() error {
	if p.Mime == "" {
		p.Mime = MIME_JSON
	}

	if err := p.prepareParam(); err != nil {
		return err
	}

	if err := p.prepareBody(); err != nil {
		return err
	}

	if p.ApiKey != nil {
		p.header.Set("X-API-Key", *p.ApiKey)
	}

	if p.Bearer != nil {
		p.header.Set("Authorization", "Bearer "+*p.Bearer)
	}

	if p.User != nil && p.Pwd != nil {
		p.header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(*p.User+":"+*p.Pwd)))
	}

	if p.header.Get("Accept") == "" {
		p.header.Set("Accept", "*/*")
	}

	if p.Client.Transport == nil {
		var err error
		if p.Client.Transport, err = p.Transport(); err != nil {
			return err
		}
	}

	if filePath := strings.TrimSpace(util.StringValue(p.OutputFile)); filePath != "" {
		if filePath == "-" {
			p.responseWriter = os.Stdout
		} else {
			fd, err := os.OpenFile(*p.OutputFile, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			p.responseWriter = fd
			p.responseCloser = fd
		}
	}

	return nil
}

func (p Request) String() string {
	return util.Prettify(p.RequestOptions)
}

func (p *Request) prepareParam() (err error) {
	// p.url, p.header, err = prepareParam(p.Url, p.header, p.InputParam)

	param := &requestParam{
		header: p.header,
		path:   map[string]string{},
		query:  map[string][]string{},
	}

	if p.InputParam == nil {
		p.url = p.Url
		return
	}

	// precheck
	if v, ok := p.InputParam.(Validator); ok {
		if err = v.Validate(); err != nil {
			klog.V(1).Infof("%s.Validate() err: %s",
				reflect.TypeOf(p.InputParam), err)
			return
		}
	}

	rv := reflect.Indirect(reflect.ValueOf(p.InputParam))
	rt := rv.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		err = fmt.Errorf("rest-encode: input must be a struct, got %v/%v", rv.Kind(), rt)
		return
	}

	fields := cachedTypeFields(rt)
	if err = param.getFromFields(rv, fields); err != nil {
		panic(err)
	}

	// gen url
	var newUrl string
	var u *url.URL
	if newUrl, err = invokePathVariable(p.Url, param.path); err != nil {
		return
	}

	if u, err = url.Parse(newUrl); err != nil {
		return
	}

	v := u.Query()
	for k1, v1 := range param.query {
		for _, v2 := range v1 {
			v.Add(k1, v2)
		}
	}
	u.RawQuery = v.Encode()

	p.url = u.String()

	return nil
}

func (p *Request) prepareBody() error {
	if filePath := strings.TrimSpace(util.StringValue(p.InputFile)); filePath != "" {
		if filePath == "-" {
			b, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			p.InputContent = b
		} else {
			info, err := os.Stat(*p.InputFile)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return fmt.Errorf("%s is dir", *p.InputFile)
			}

			fd, err := os.Open(*p.InputFile)
			if err != nil {
				return err
			}
			p.bodyReader = fd
			p.bodyCloser = fd
			p.header.Set("Content-Length", fmt.Sprintf("%d", info.Size()))

			return nil
		}
	}

	if len(p.InputContent) > 0 {
		p.header.Set("Content-Type", p.Mime)
		p.bodyContent = p.InputContent
		p.bodyReader = bytes.NewReader(p.bodyContent)
		p.header.Set("Content-Length", fmt.Sprintf("%d", len(p.bodyContent)))
		return nil
	}

	if body := p.InputBody; body != nil {
		var err error
		switch p.Mime {
		case MIME_JSON:
			if p.bodyContent, err = json.Marshal(body); err != nil {
				return err
			}
		case MIME_XML:
			if p.bodyContent, err = xml.Marshal(body); err != nil {
				return err
			}
		case MIME_URL_ENCODED:
			if p.bodyContent, err = urlencoded.Marshal(body); err != nil {
				return err
			}
		default:
			return fmt.Errorf("http request header Content-Type invalid " + p.Mime)
		}

		if p.Method != "GET" {
			p.header.Set("Content-Type", p.Mime)
			p.bodyReader = bytes.NewReader(p.bodyContent)
			p.header.Set("Content-Length", fmt.Sprintf("%d", len(p.bodyContent)))
		}
		return nil
	}

	return nil
}

func (p *Request) Content() []byte {
	return p.bodyContent
}

func (p *Request) HeaderSet(key, value string) {
	p.header.Set(key, value)
}

func (p *Request) Do() (resp *http.Response, err error) {
	var respBody []byte
	r := p.Request

	defer func() {
		if !klog.V(5).Enabled() {
			return
		}

		body := p.bodyContent
		if len(body) > 1024 {
			body = body[:1024]
		}
		klog.Infof("[req] %s", Req2curl(r, body, p.InputFile, p.OutputFile))

		buf := &bytes.Buffer{}
		HttpRespPrint(buf, resp, respBody)
		if buf.Len() > 0 {
			klog.Infof(buf.String())
		}
	}()

	// ctx & tracer
	if sp := opentracing.SpanFromContext(p.Ctx); sp != nil {
		p.Client.Transport = &nethttp.Transport{}

		r = r.WithContext(p.Ctx)
		var ht *nethttp.Tracer
		r, ht = nethttp.TraceRequest(sp.Tracer(), r)
		defer ht.Finish()
	}

	if resp, err = p.Client.Do(r); err != nil {
		return
	}

	defer func() {
		if p.bodyCloser != nil {
			p.bodyCloser.Close()
		}
		if p.responseCloser != nil {
			p.responseCloser.Close()
		}
		resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		respBody, _ = ioutil.ReadAll(resp.Body)
		err = fmt.Errorf("%d: %s", resp.StatusCode, respBody)
		return
	}

	if p.responseWriter != nil {
		_, err = io.Copy(p.responseWriter, resp.Body)
		return
	}

	if out, ok := p.Output.(io.Writer); ok {
		_, err = io.Copy(out, resp.Body)
		return
	}

	if respBody, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}

	if p.Output == nil {
		return
	}

	switch mime := resp.Header.Get("Content-Type"); mime {
	case MIME_XML:
		err = xml.Unmarshal(respBody, p.Output)
	case MIME_URL_ENCODED:
		err = urlencoded.Unmarshal(respBody, p.Output)
	case MIME_JSON:
		err = json.Unmarshal(respBody, p.Output)
	default:
		err = json.Unmarshal(respBody, p.Output)
	}

	return
}

func (p *Request) Curl() string {
	return Req2curl(p.Request, p.bodyContent, p.InputFile, p.OutputFile)
}

// dst: must be ptr
func ReadEntity(req *restful.Request, param, body interface{}) error {
	ctx := WithParam(req.Request.Context(), param)
	ctx = WithBody(ctx, body)
	req.Request = req.Request.WithContext(ctx)

	return readEntity(req, param, body)
}

// Request -> struct
func readEntity(r *restful.Request, param, body interface{}) error {
	p := &requestParam{
		header: r.Request.Header,
		path:   r.PathParameters(),
		query:  r.Request.URL.Query(),
	}

	rv := reflect.ValueOf(param)
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

	// param
	fields := cachedTypeFields(rt)
	for _, f := range fields.list {
		subv, err := getSubv(rv, f.index, true)
		if err != nil {
			return err
		}
		if err := p.setFieldValue(&f, subv); err != nil {
			return err
		}
	}

	// body
	if body != nil {
		if err := r.ReadEntity(body); err != nil {
			klog.V(5).Infof("restful.ReadEntity() error %s", err)
			return err
		}
	}

	// postcheck
	if v, ok := param.(Validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	if v, ok := body.(Validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}
