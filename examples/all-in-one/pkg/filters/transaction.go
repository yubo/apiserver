package filters

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/golib/orm"
	"k8s.io/klog/v2"
)

func WithTx(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	ctx := req.Request.Context()
	tx, err := models.DB().BeginTx(ctx, nil)
	if err != nil {
		responsewriters.InternalError(resp, req.Request, fmt.Errorf("BeginTX err %s", err))
		return
	}

	req.Request.WithContext(orm.WithDB(ctx, tx))

	chain.ProcessFilter(req, resp)

	// setAttribute("error", err) by rest.registerHandle
	if err, ok := req.Attribute("error").(error); ok && err != nil {
		if err := tx.Rollback(); err != nil {
			klog.ErrorS(err, "rollback")
		}
	} else {
		if err := tx.Commit(); err != nil {
			klog.ErrorS(err, "commit")
		}
	}

}
