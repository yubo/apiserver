package main

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/rest"
	jose "gopkg.in/square/go-jose.v2"
	"k8s.io/klog/v2"

	// http
	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

var (
	issuerURL string
)

const (
	moduleName = "oidc-provider"
)

type config struct {
	RSAKey       string `json:"rsakey"`
	OpenIDConfig string `json:"openIDConfig"`
	Claim        string `json:"claim"`
}

func newConfig() *config {
	return &config{}
}

type module struct {
	name         string
	pubKeys      []*jose.JSONWebKey
	webKeySet    jose.JSONWebKeySet
	signingKey   *jose.JSONWebKey
	openIDConfig string
	claim        string
}

var (
	_module = &module{name: moduleName}
	hookOps = []v1.HookOps{{
		Hook:     _module.start,
		Owner:    moduleName,
		HookNum:  v1.ACTION_START,
		Priority: v1.PRI_MODULE,
	}}
)

func main() {
	command := proc.NewRootCmd(server.WithoutTLS())
	code := cli.Run(command)
	os.Exit(code)
}

func (p *module) start(ctx context.Context) error {
	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}

	p.pubKeys = []*jose.JSONWebKey{loadRSAKey(cf.RSAKey, jose.RS256)}
	p.webKeySet = toKeySet(p.pubKeys)
	p.signingKey = loadRSAPrivKey(cf.RSAKey, jose.RS256)
	p.openIDConfig = cf.OpenIDConfig
	p.claim = cf.Claim

	p.installWs(options.APIServerMustFrom(ctx))

	klog.InfoS("test data", "token", p.token())
	return nil
}

func toKeySet(keys []*jose.JSONWebKey) jose.JSONWebKeySet {
	ret := jose.JSONWebKeySet{}
	for _, k := range keys {
		ret.Keys = append(ret.Keys, *k)
	}
	return ret
}

func (p *module) installWs(c rest.GoRestfulContainer) {
	c.UnlistedHandle("/.testing/keys", http.HandlerFunc(p.keys))
	c.UnlistedHandle("/.well-known/openid-configuration", http.HandlerFunc(p.openidConfiguration))
}

func (p *module) keys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	keyBytes, _ := json.Marshal(p.webKeySet)
	klog.V(5).Infof("%v: returning: %+v", r.URL, string(keyBytes))
	w.Write(keyBytes)
}

func (p *module) openidConfiguration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(p.openIDConfig))
}

func (p *module) token() string {
	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.SignatureAlgorithm(p.signingKey.Algorithm),
		Key:       p.signingKey,
	}, nil)
	if err != nil {
		klog.Fatalf("initialize signer: %v", err)
	}

	jws, err := signer.Sign([]byte(fmt.Sprintf(p.claim, time.Now().Add(24*time.Hour).Unix())))
	if err != nil {
		klog.Fatalf("sign claims: %v", err)
	}

	token, err := jws.CompactSerialize()
	if err != nil {
		klog.Fatalf("serialize token: %v", err)
	}

	return token
}

func loadRSAKey(filepath string, alg jose.SignatureAlgorithm) *jose.JSONWebKey {
	return loadKey(filepath, alg, func(b []byte) (interface{}, error) {
		key, err := x509.ParsePKCS1PrivateKey(b)
		if err != nil {
			return nil, err
		}
		return key.Public(), nil
	})
}

func loadRSAPrivKey(filepath string, alg jose.SignatureAlgorithm) *jose.JSONWebKey {
	return loadKey(filepath, alg, func(b []byte) (interface{}, error) {
		return x509.ParsePKCS1PrivateKey(b)
	})
}

func loadKey(filepath string, alg jose.SignatureAlgorithm, unmarshal func([]byte) (interface{}, error)) *jose.JSONWebKey {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalf("load file: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		log.Fatalf("file contained no PEM encoded data: %s", filepath)
	}
	priv, err := unmarshal(block.Bytes)
	if err != nil {
		log.Fatalf("unmarshal key: %v", err)
	}
	key := &jose.JSONWebKey{Key: priv, Use: "sig", Algorithm: string(alg)}
	thumbprint, err := key.Thumbprint(crypto.SHA256)
	if err != nil {
		log.Fatalf("computing thumbprint: %v", err)
	}
	key.KeyID = hex.EncodeToString(thumbprint)
	return key
}

func init() {
	// register hookOps as a module
	proc.RegisterHooks(hookOps)

	// register config{} to configer.Factory
	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("oidc-provider"))
}
