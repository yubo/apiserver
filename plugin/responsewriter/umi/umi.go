package umi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/apiserver/pkg/rest"
)

// https://pro.ant.design/zh-CN/docs/request

var (
	_hostname, _ = os.Hostname()
	RespWriter   = newRespWriter()
	schemas      = map[int]string{
		http.StatusOK: `{
  "required": [
    "success"
  ],
  "properties": {
    "success": {
      "type": "boolean"
    },
    "host": {
      "type": "string"
    },
    "traceId": {
      "type": "string"
    }
  }
}`,

		http.StatusBadRequest: `{
  "required": [
    "errorCode",
    "errorMessage",
    "success"
  ],
  "properties": {
    "errorCode": {
      "type": "string"
    },
    "errorMessage": {
      "type": "string"
    },
    "success": {
      "type": "boolean"
    }
  }
}`,
	}
)

type route struct {
	method string
	path   string
}

func newRespWriter() rest.RespWriter {
	return &respWriter{
		routes:  []route{},
		schemas: schemas,
	}
}

type respWriter struct {
	routes  []route
	schemas map[int]string
}

// Name: use to register
func (p *respWriter) Name() string {
	return "umi.respwriter"
}

// RespWrite: use to customize response output format
// https://pro.ant.design/zh-CN/docs/request
func (p *respWriter) RespWrite(resp *restful.Response, req *http.Request, data interface{}, err error) {
	v := map[string]interface{}{
		"success": true,
		"host":    _hostname,
	}

	if traceID, _ := request.TraceIDFrom(req.Context()); traceID != "" {
		v["traceId"] = traceID
	}

	if data != nil {
		v["data"] = data
	}

	if err != nil {
		v["success"] = false
		v["errorMessage"] = err.Error()
		v["errorCode"] = strconv.Itoa(int(responsewriters.ErrorToAPIStatus(err).Code))
	}

	resp.WriteEntity(v)
}

// SwaggerHandler: called at PostBuildSwaggerObjectHandler, use to rewrite the response definitions
func (p *respWriter) SwaggerHandler(s *spec.Swagger) {
	for _, route := range p.routes {
		o, err := rest.OperationFrom(s, route.method, route.path)
		if err != nil {
			panic(err)
		}

		for status, v := range p.schemas {
			definition := spec.Schema{}
			if err := json.Unmarshal([]byte(v), &definition); err != nil {
				panic(err)
			}

			modelType := "_umi_"

			// v is a copy of schema
			resp, ok := o.Responses.StatusCodeResponses[status]
			if !ok {
				resp = spec.Response{}
				resp.Description = http.StatusText(status)
			} else {
				if name := nameOfSchema(resp.Schema); strings.HasPrefix(name, "_umi_") {
					panic(fmt.Sprintf("method %s:%s invalie prefix model name %s", route.method, route.path, name))
				} else {
					modelType = "_umi_." + name
				}
				definition.Properties["data"] = *resp.Schema
			}

			{
				var schema spec.Schema
				schema.Ref = spec.MustCreateRef("#/definitions/" + modelType)
				resp.Schema = &schema
			}

			o.Responses.StatusCodeResponses[status] = resp

			if _, ok := s.Definitions[modelType]; !ok {
				s.Definitions[modelType] = definition
			}
		}
	}
}

// AddRoute: register route
func (p *respWriter) AddRoute(method, path string) {
	p.routes = append(p.routes, route{method: method, path: path})
}

func init() {
	rest.ResponseWriterRegister(RespWriter)
}

func nameOfSchema(prop *spec.Schema) string {
	if prop == nil {
		return ""
	}

	// ref
	if u := prop.Ref.GetURL(); u != nil {
		// just for go-restful-openapi
		return strings.TrimPrefix(u.String(), "#/definitions/")
	}

	if len(prop.Type) == 0 {
		panic("type is not set")
	}

	switch t := prop.Type[0]; t {
	case "array":
		if len(prop.Items.Schema.Type) == 0 || prop.Items.Schema.Type[0] == "" {
			panic("array's type is not set")
		}
		return t + "." + prop.Items.Schema.Type[0]
	case "integer", "number", "boolean", "string":
		return t
	default:
		panic(fmt.Sprintf("unsupported type %s", t))
	}
}
