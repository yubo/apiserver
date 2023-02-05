package v1

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

var InvalidVersionErr = errors.New("can not get version infomation")

type Hooks []*HookOps

func (x Hooks) Sort() {
	sort.SliceStable(x, func(i, j int) bool { return x[i].ProcessPriority() < x[j].ProcessPriority() })
}

type HookFn func(context.Context) error

type HookOps struct {
	Hook        HookFn
	Owner       string
	HookNum     ProcessAction
	Priority    uint16
	SubPriority uint16
	Data        interface{}

	processPriority ProcessPriority
	ctx             context.Context
	//process  *Process
}

func (p HookOps) SetContext(ctx context.Context) {
	p.ctx = ctx
}

func (p HookOps) Context() context.Context {
	return p.ctx
}

func (p *HookOps) ProcessPriority() ProcessPriority {
	if p.processPriority == 0 {
		p.processPriority = ProcessPriority(uint32(p.Priority)<<(16-3) + uint32(p.SubPriority))
	}
	return p.processPriority
}

//
//func (p HookOps) Configer() configer.ParsedConfiger {
//	return p.process.parsedConfiger
//}

//func (p HookOps) ContextAndConfiger() (context.Context, configer.ParsedConfiger) {
//	return p.Context(), p.Configer()
//}

func (p HookOps) Dlog() {
	if klog.V(5).Enabled() {
		klog.InfoSDepth(1, "dispatch hook",
			"hookName", p.HookNum.String(),
			"owner", p.Owner,
			"priority", p.processPriority.String(),
			"nameOfFunction", util.Name(p.Hook))
	}
}

type ProcessPriority uint32

const (
	_                 uint16 = iota << 3
	PRI_SYS_INIT             // init & register each system.module
	PRI_SYS_PRESTART         // prepare each system.module's depend
	PRI_MODULE               // init each module
	PRI_SYS_START            // start each system.module
	PRI_SYS_POSTSTART        // no use
)

const (
	_ uint16 = iota << 3

	PRI_M_LOGGING // no dep
	PRI_M_DB      // no  dep
	PRI_M_AUDIT   // no  dep
	PRI_M_HTTP    // no dep
	PRI_M_AUTHN   // dep authn_mode HTTP1
	PRI_M_AUTHZ   // dep authn
	PRI_M_HTTP2   // dep authn authz audit
	PRI_M_TRACING // dep HTTP2
	PRI_M_GRPC    // dep tracing authn authz audit
)

func (p ProcessPriority) String() string {
	return fmt.Sprintf("0x%08x", uint32(p))
}

type ProcessAction uint32

const (
	ACTION_START ProcessAction = iota
	ACTION_RELOAD
	ACTION_STOP
	ACTION_SIZE
)

type ProcessStatus uint32

const (
	STATUS_INIT ProcessStatus = iota
	STATUS_PENDING
	STATUS_RUNNING
	STATUS_RELOADING
	STATUS_EXIT
)

func (p *ProcessStatus) Set(v ProcessStatus) {
	atomic.StoreUint32((*uint32)(p), uint32(STATUS_RUNNING))
}

func (p ProcessAction) String() string {
	switch p {
	case ACTION_START:
		return "start"
	case ACTION_RELOAD:
		return "reload"
	case ACTION_STOP:
		return "stop"
	default:
		return "unknown"
	}

}
