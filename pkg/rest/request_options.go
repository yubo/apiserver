package rest

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/yubo/golib/crypto/tlsutil"
	"github.com/yubo/golib/net/urlutil"
	"github.com/yubo/golib/util"
)

type RequestOptions struct {
	http.Client
	Url                string // https://example.com/api/v{version}/{model}/{object}?type=vm
	Method             string
	User               *string
	Pwd                *string
	Bearer             *string
	Token              *string
	TokenField         *string
	ApiKey             *string
	InputParam         interface{}
	InputContent       []byte
	InputFile          *string // Priority InputFile > InputContent > InputBody
	InputBody          interface{}
	OutputFile         *string // Priority OutputFile > Output
	Output             interface{}
	Mime               string
	Ctx                context.Context
	header             http.Header
	CertFile           string
	KeyFile            string
	CaFile             string
	InsecureSkipVerify bool
}

func (p RequestOptions) String() string {
	return util.Prettify(p)
}

func (p RequestOptions) Transport() (tr *http.Transport, err error) {
	tr = &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
	}
	if (p.CertFile != "" && p.KeyFile != "") || p.CaFile != "" {
		tlsConf, err := p.TLSClientConfig()
		if err != nil {
			return nil, fmt.Errorf("can't create TLS config: %s", err.Error())
		}
		tr.TLSClientConfig = tlsConf
	}
	return tr, nil
}

func (p RequestOptions) TLSClientConfig() (*tls.Config, error) {
	serverName, err := urlutil.ExtractHostname(p.Url)
	if err != nil {
		return nil, err
	}

	return tlsutil.ClientConfig(tlsutil.Options{
		CaCertFile:         p.CaFile,
		KeyFile:            p.KeyFile,
		CertFile:           p.CertFile,
		InsecureSkipVerify: p.InsecureSkipVerify,
		ServerName:         serverName,
	})

}

type RequestOption interface {
	apply(*RequestOptions)
}

type funcRequestOption struct {
	f func(*RequestOptions)
}

func (p *funcRequestOption) apply(opt *RequestOptions) {
	p.f(opt)
}

func newFuncRequestOption(f func(*RequestOptions)) *funcRequestOption {
	return &funcRequestOption{
		f: f,
	}
}

func WithUrl(url string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.Url = url
	})
}

func WithMethod(method string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.Method = method
	})
}

func WithBase(user, pwd string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.User = &user
		o.Pwd = &pwd
	})
}

func WithBearer(bearer string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.Bearer = &bearer
	})
}

func WithToken(token string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.Token = &token
	})
}
func WithTokenField(tokenField string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.TokenField = &tokenField
	})
}
func WithApiKey(apiKey string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.ApiKey = &apiKey
	})
}

func WithInputFile(filePath string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.InputFile = &filePath
	})
}

func WithInputContent(body []byte) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.InputContent = body
	})
}

func WithInputParam(param interface{}) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.InputParam = param
	})
}

func WithInputBody(body interface{}) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.InputBody = body
	})
}

func WithOutputFile(filePath string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.OutputFile = &filePath
	})
}

func WithOutput(output interface{}) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.Output = output
	})
}

func WithMime(mime string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.Mime = mime
	})
}

func WithCtx(ctx context.Context) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.Ctx = ctx
	})
}

func WithHeader(k, v string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.header.Set(k, v)
	})
}

func WithTLSConfig(certFile, keyFile, caFile string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.CertFile = certFile
		o.KeyFile = keyFile
		o.CaFile = caFile
	})
}

func InsecureSkipVerify() RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.InsecureSkipVerify = true
	})
}
