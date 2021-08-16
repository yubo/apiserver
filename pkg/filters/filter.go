package filters

import (
	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/golib/runtime/serializer"
)

func HttpFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	ctx := req.Request.Context()
	ctx = request.WithResp(ctx, resp)

	req.Request = req.Request.WithContext(ctx)

	chain.ProcessFilter(req, resp)
}

var Codecs = serializer.NewCodecFactory()
