package cmdcli

import (
	"context"
	"io"
	"time"

	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/scheme"
)

// host: 127.0.0.1:8080
func NewRequest(host string, opts ...RequestOption) (*Request, error) {
	config := &rest.Config{
		Host: host,
		ContentConfig: rest.ContentConfig{
			NegotiatedSerializer: scheme.Codecs,
		},
	}
	return NewRequestWithConfig(config)
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

	if p.prefix != "" {
		req = req.Prefix(p.prefix)
	}

	if p.param != nil {
		req = req.VersionedParams(p.param, p.client.Codec())
	}
	if p.body != nil {
		req = req.Body(p.body)
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
	client  *rest.RESTClient    // client
	timeout time.Duration       // second
	param   interface{}         // param variables
	body    interface{}         // string, []byte, io.Reader, struct{}
	output  interface{}         //
	cb      []func(interface{}) //
}

type RequestOption func(*RequestOptions)

func WithClient(client *rest.RESTClient) RequestOption {
	return func(o *RequestOptions) {
		o.client = client
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
func WithParam(param interface{}) RequestOption {
	return func(o *RequestOptions) {
		o.param = param
	}
}
func WithBody(body interface{}) RequestOption {
	return func(o *RequestOptions) {
		o.body = body
	}
}
func WithOutput(output interface{}) RequestOption {
	return func(o *RequestOptions) {
		o.output = output
	}
}
func WithCallback(cb ...func(interface{})) RequestOption {
	return func(o *RequestOptions) {
		o.cb = cb
	}
}
func WithTimeout(timeout time.Duration) RequestOption {
	return func(o *RequestOptions) {
		o.timeout = timeout
	}
}
