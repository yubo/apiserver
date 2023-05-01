package client

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/yubo/apiserver/pkg/scheme"
	"github.com/yubo/client-go/rest"
)

// host: http://127.0.0.1:8080
func NewRequest(host string, opts ...RequestOption) (*Request, error) {
	config := &rest.Config{
		Host: host,
		ContentConfig: rest.ContentConfig{
			NegotiatedSerializer: scheme.Codecs,
		},
	}
	return NewRequestWithConfig(config, opts...)
}

func NewRequestWithConfig(config *rest.Config, opts ...RequestOption) (*Request, error) {
	client, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}
	return NewRequestWithClient(client, opts...), nil
}

func NewRequestWithClient(client *rest.RESTClient, opts ...RequestOption) *Request {
	o := &Request{
		client: client,
	}

	// default method
	o.method = "GET"

	for _, opt := range opts {
		opt(&o.RequestOptions)
	}
	return o
}

func (p *Request) Pager(stdout io.Writer, disablePage bool) *Pager {
	pager, err := NewPager(p, stdout, disablePage)
	if err != nil {
		panic(err)
	}
	return pager
}

type Request struct {
	client *rest.RESTClient
	RequestOptions
}

// ("GET", "https://example.com/api/v{version}/{model}/{subject}?a=1&b=2", {"subject":"abc", "model": "instance", "version": 1}, nil)
func (p *Request) Do(ctx context.Context) error {
	req := p.client.Verb(p.method)

	if p.httpClient != nil {
		p.client.Client = p.httpClient
	}

	if p.prefix != "" {
		req = req.Prefix(p.prefix)
	}

	if p.debug {
		req = req.Debug()
	}

	if p.param != nil {
		req = req.VersionedParams(p.param, scheme.ParameterCodec)
	}
	if p.body != nil {
		req = req.Body(p.body)
	}
	for k, v := range p.header {
		req.SetHeader(k, v...)
	}

	if p.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	if w, ok := p.output.(io.Writer); ok {
		b, err := req.Do(ctx).Raw()
		if err != nil {
			return err
		}

		if _, err := w.Write(b); err != nil {
			return err
		}
		w.Write([]byte("\n"))
		return nil
	}

	if err := req.Do(ctx).Into(p.output); err != nil {
		return err
	}

	for _, fn := range p.cb {
		if fn != nil {
			fn(p.output)
		}
	}

	return nil
}

type RequestOptions struct {
	method  string
	prefix  string
	debug   bool
	header  http.Header
	timeout time.Duration       // second
	param   interface{}         // param variables,
	body    interface{}         // string, []byte, io.Reader, struct{}
	output  interface{}         // io.Writer, struct{}
	cb      []func(interface{}) // callback after req.Do()

	// Set specific behavior of the client.  If not set http.DefaultClient will be used.
	httpClient *http.Client
}

type RequestOption func(*RequestOptions)

func WithHeader(header http.Header) RequestOption {
	return func(o *RequestOptions) {
		if o.header == nil {
			o.header = header
			return
		}
		for k, values := range header {
			for _, v := range values {
				o.header.Add(k, v)
			}
		}
	}
}

func WithClient(client *http.Client) RequestOption {
	return func(o *RequestOptions) {
		o.httpClient = client
	}
}
func WithMethod(method string) RequestOption {
	return func(o *RequestOptions) {
		o.method = method
	}
}
func WithPrefix(prefix string) RequestOption {
	return func(o *RequestOptions) {
		o.prefix = prefix
	}
}
func WithPath(path string) RequestOption {
	return func(o *RequestOptions) {
		o.prefix = path
	}
}
func WithDebug() RequestOption {
	return func(o *RequestOptions) {
		o.debug = true
	}
}

// WithParams: encode by request.ParameterCodec for req.{HEAD, Param, Path}
func WithParams(param interface{}) RequestOption {
	return func(o *RequestOptions) {
		o.param = param
	}
}

// WithBody makes the request use obj as the body. Optional.
// If obj is a string, try to read a file of that name.
// If obj is a []byte, send it directly.
// If obj is an io.Reader, use it directly.
// If obj is a runtime.Object, marshal it correctly, and set Content-Type header.
// If obj is a runtime.Object and nil, do nothing.
// Otherwise, set an error.
func WithBody(obj interface{}) RequestOption {
	return func(o *RequestOptions) {
		o.body = obj
	}
}

// WithOutput: output.(io.Writer) or decode.into(output)
func WithOutput(output interface{}) RequestOption {
	return func(o *RequestOptions) {
		o.output = output
	}
}
func WithCallback(cb ...func(output interface{})) RequestOption {
	return func(o *RequestOptions) {
		o.cb = cb
	}
}
func WithTimeout(timeout time.Duration) RequestOption {
	return func(o *RequestOptions) {
		o.timeout = timeout
	}
}
