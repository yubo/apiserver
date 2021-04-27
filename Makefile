TARGET= modules/authentication/options/zz_generated.deepcopy.go \
	modules/authorization/zz_generated.deepcopy.go \
	modules/audit/zz_generated.deepcopy.go \
	modules/secure/options/zz_generated.deepcopy.go \
	pkg/server/options/zz_generated.deepcopy.go

all: $(TARGET)

modules/authentication/options/zz_generated.deepcopy.go: 
	deepcopy-gen --input-dirs ./modules/authentication/options --output-package github.com/yubo/apiserver/modules/authentication/options -O zz_generated.deepcopy

modules/secure/options/zz_generated.deepcopy.go: 
	deepcopy-gen --input-dirs ./modules/secure/options --output-package github.com/yubo/apiserver/modules/secure/options -O zz_generated.deepcopy

modules/authorization/zz_generated.deepcopy.go: 
	deepcopy-gen --input-dirs ./modules/authorization --output-package github.com/yubo/apiserver/modules/authorization -O zz_generated.deepcopy

modules/audit/zz_generated.deepcopy.go: 
	deepcopy-gen --input-dirs ./modules/audit --output-package github.com/yubo/apiserver/modules/audit -O zz_generated.deepcopy

pkg/server/options/zz_generated.deepcopy.go: 
	deepcopy-gen --input-dirs ./pkg/server/options --output-package github.com/yubo/apiserver/pkg/server/optioins -O zz_generated.deepcopy



