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
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/errors"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
)

// https://pro.ant.design/zh-CN/docs/request

var (
	ModelTypePrefix = "__umi__."
	Hostname, _     = os.Hostname()
	RespWriter      = newRespWriter()
	schemas         = map[int]string{
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

func newRespWriter() server.RespWriter {
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
func (p *respWriter) RespWrite(resp *restful.Response, req *http.Request, data interface{}, err error, s runtime.NegotiatedSerializer) {
	v := map[string]interface{}{
		"success": true,
		"host":    Hostname,
	}

	if traceID := trace.SpanFromContext(req.Context()).SpanContext().TraceID(); traceID.IsValid() {
		v["traceId"] = traceID.String()
	}

	if data != nil {
		v["data"] = data
	}

	if err != nil {
		v["success"] = false
		v["errorMessage"] = err.Error()
		v["errorCode"] = strconv.Itoa(int(responsewriters.ErrorToAPIStatus(err).Code))
	}

	//	resp.WriteEntity(v)
	responsewriters.WriteObjectNegotiated(s, resp.ResponseWriter, req, 200, v)
}

// SwaggerHandler: called at PostBuildSwaggerObjectHandler, use to rewrite the response definitions
func (p *respWriter) SwaggerHandler(s *spec.Swagger) {
	for _, route := range p.routes {
		o, err := server.OperationFrom(s, route.method, route.path)
		if err != nil {
			panic(err)
		}

		for status, schema := range p.schemas {
			resp, modelType, rawSchema, err := buildResponse(status, o.Responses)
			if err != nil {
				klog.ErrorS(err, "swaggerHandler.buildResponse", "method", route.method, "path", route.path)
				panic(err)
			}

			o.Responses.StatusCodeResponses[status] = resp

			if _, ok := s.Definitions[modelType]; !ok {
				definition := spec.Schema{}
				if err := json.Unmarshal([]byte(schema), &definition); err != nil {
					panic(err)
				}

				if rawSchema != nil {
					definition.Properties["data"] = *rawSchema
				}

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
	server.ResponseWriterRegister(RespWriter)
}

func nameOfSchema(status int, prop *spec.Schema) string {
	if prop == nil {
		return "Response" + strconv.Itoa(status)
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
		if u := prop.Items.Schema.Ref.GetURL(); u != nil {
			return t + "." + strings.TrimPrefix(u.String(), "#/definitions/")
		}
		if len(prop.Items.Schema.Type) > 0 && prop.Items.Schema.Type[0] != "" {
			return t + "." + prop.Items.Schema.Type[0]
		}
		panic(fmt.Sprintf("invalid array type %s", util.JsonStr(prop)))
	case "integer", "number", "boolean", "string":
		return t
	default:
		panic(fmt.Sprintf("unsupported type %s", util.JsonStr(prop)))
	}
}

func buildResponse(status int, responses *spec.Responses) (resp spec.Response, modelType string, rawSchema *spec.Schema, err error) {
	var ok bool
	if resp, ok = responses.StatusCodeResponses[status]; !ok {
		resp = spec.Response{}
		resp.Description = http.StatusText(status)
	}

	modelType = nameOfSchema(status, resp.Schema)
	rawSchema = resp.Schema

	if strings.HasPrefix(modelType, ModelTypePrefix) {
		err = errors.Errorf("invalie prefix model name %s", modelType)
		return
	}
	modelType = ModelTypePrefix + modelType

	schema := spec.Schema{}
	schema.Ref = spec.MustCreateRef("#/definitions/" + modelType)
	resp.Schema = &schema

	return
}
