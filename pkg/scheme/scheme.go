package scheme

import (
	"github.com/yubo/apiserver/pkg/server/parametercodec"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/runtime/serializer"
)

// deprecated
var (
	Codecs  = serializer.NewCodecFactory()
	Codec   = Codecs.LegacyCodec()
	Decoder = Codecs.UniversalDeserializer()

	// multiple SerializerInfo
	NegotiatedSerializer = Codecs.WithoutConversion()

	ClientNegotiator = runtime.NewClientNegotiator(NegotiatedSerializer)

	ParameterCodec = parametercodec.New()
)
