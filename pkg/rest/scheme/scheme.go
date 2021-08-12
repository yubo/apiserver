package scheme

import (
	"github.com/yubo/apiserver/staging/runtime"
	"github.com/yubo/apiserver/staging/runtime/serializer"
)

var Codecs = serializer.NewCodecFactory()
var ParameterCodec = runtime.NewParameterCodec()
