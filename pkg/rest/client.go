package rest

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yubo/golib/api"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/util/flowcontrol"
)

// Interface captures the set of operations for generically interacting with Kubernetes REST apis.
type Interface interface {
	GetRateLimiter() flowcontrol.RateLimiter
	Verb(verb string) *Request
	Post() *Request
	Put() *Request
	Patch(pt api.PatchType) *Request
	Get() *Request
	Delete() *Request
}

// ClientContentConfig controls how RESTClient communicates with the server.
//
// TODO: ContentConfig will be updated to accept a Negotiator instead of a
//   NegotiatedSerializer and NegotiatedSerializer will be removed.
type ClientContentConfig struct {
	// AcceptContentTypes specifies the types the client will accept and is optional.
	// If not set, ContentType will be used to define the Accept header
	AcceptContentTypes string
	// ContentType specifies the wire format used to communicate with the server.
	// This value will be set as the Accept header on requests made to the server if
	// AcceptContentTypes is not set, and as the default content type on any object
	// sent to the server. If not set, "application/json" is used.
	ContentType string
	// Negotiator is used for obtaining encoders and decoders for multiple
	// supported media types.
	Negotiator runtime.ClientNegotiator

	BackoffBase     int
	BackoffDuration int
}

// RESTClient imposes common Kubernetes API conventions on a set of resource paths.
// The baseURL is expected to point to an HTTP or HTTPS path that is the parent
// of one or more resources.  The server should return a decodable API resource
// object, or an api.Status object which contains information about the reason for
// any failure.
//
// Most consumers should use client.New() to get a Kubernetes API client.
type RESTClient struct {
	// base is the root URL for all invocations of the client
	base *url.URL

	// content describes how a RESTClient encodes and decodes responses.
	content ClientContentConfig

	// creates BackoffManager that is passed to requests.
	createBackoffMgr func() BackoffManager

	// rateLimiter is shared among all requests created by this client unless specifically
	// overridden.
	rateLimiter flowcontrol.RateLimiter

	// warningHandler is shared among all requests created by this client.
	// If not set, defaultWarningHandler is used.
	warningHandler WarningHandler

	// Set specific behavior of the client.  If not set http.DefaultClient will be used.
	Client *http.Client
}

// NewRESTClient creates a new RESTClient. This client performs generic REST functions
// such as Get, Put, Post, and Delete on specified paths.
func NewRESTClient(baseURL *url.URL, config ClientContentConfig, rateLimiter flowcontrol.RateLimiter, client *http.Client) (*RESTClient, error) {
	if len(config.ContentType) == 0 {
		config.ContentType = "application/json"
	}

	base := *baseURL
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	base.RawQuery = ""
	base.Fragment = ""

	return &RESTClient{
		base:             &base,
		content:          config,
		createBackoffMgr: config.readExpBackoffConfig,
		rateLimiter:      rateLimiter,

		Client: client,
	}, nil
}

// GetRateLimiter returns rate limiter for a given client, or nil if it's called on a nil client
func (c *RESTClient) GetRateLimiter() flowcontrol.RateLimiter {
	if c == nil {
		return nil
	}
	return c.rateLimiter
}

// readExpBackoffConfig handles the internal logic of determining what the
// backoff policy is.  By default if no information is available, NoBackoff.
// TODO Generalize this see #17727 .
func (p ClientContentConfig) readExpBackoffConfig() BackoffManager {
	if p.BackoffBase <= 0 || p.BackoffDuration <= 0 {
		return &NoBackoff{}
	}
	return &URLBackoff{
		Backoff: flowcontrol.NewBackOff(
			time.Duration(p.BackoffBase)*time.Second,
			time.Duration(p.BackoffDuration)*time.Second)}
}

// Verb begins a request with a verb (GET, POST, PUT, DELETE).
//
// Example usage of RESTClient's request building interface:
// c, err := NewRESTClient(...)
// if err != nil { ... }
// resp, err := c.Verb("GET").
//  Path("pods").
//  SelectorParam("labels", "area=staging").
//  Timeout(10*time.Second).
//  Do()
// if err != nil { ... }
// list, ok := resp.(*api.PodList)
//
func (c *RESTClient) Verb(verb string) *Request {
	return NewRequest(c).Verb(verb)
}

// Post begins a POST request. Short for c.Verb("POST").
func (c *RESTClient) Post() *Request {
	return c.Verb("POST")
}

// Put begins a PUT request. Short for c.Verb("PUT").
func (c *RESTClient) Put() *Request {
	return c.Verb("PUT")
}

// Patch begins a PATCH request. Short for c.Verb("Patch").
func (c *RESTClient) Patch(pt api.PatchType) *Request {
	return c.Verb("PATCH").SetHeader("Content-Type", string(pt))
}

// Get begins a GET request. Short for c.Verb("GET").
func (c *RESTClient) Get() *Request {
	return c.Verb("GET")
}

// Delete begins a DELETE request. Short for c.Verb("DELETE").
func (c *RESTClient) Delete() *Request {
	return c.Verb("DELETE")
}
