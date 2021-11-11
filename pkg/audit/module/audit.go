package module

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	auditinternal "github.com/yubo/apiserver/pkg/apis/audit"
	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/audit/policy"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/util/webhook"
	pluginbuffered "github.com/yubo/apiserver/plugin/audit/buffered"
	pluginlog "github.com/yubo/apiserver/plugin/audit/log"
	plugintruncate "github.com/yubo/apiserver/plugin/audit/truncate"
	pluginwebhook "github.com/yubo/apiserver/plugin/audit/webhook"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util/errors"
	utilnet "github.com/yubo/golib/util/net"
	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/klog/v2"
)

const (
	moduleName = "audit"

	// Default configuration values for ModeBatch.
	defaultBatchBufferSize = 10000 // Buffer up to 10000 events before starting discarding.
	// These batch parameters are only used by the webhook backend.
	defaultBatchMaxSize       = 400              // Only send up to 400 events at a time.
	defaultBatchMaxWait       = 30 * time.Second // Send events at least twice a minute.
	defaultBatchThrottleQPS   = 10               // Limit the send rate by 10 QPS.
	defaultBatchThrottleBurst = 15               // Allow up to 15 QPS burst.
)

func appendBackend(existing, newBackend audit.Backend) audit.Backend {
	if existing == nil {
		return newBackend
	}
	if newBackend == nil {
		return existing
	}
	return audit.Union(existing, newBackend)
}

const (
	// ModeBatch indicates that the audit backend should buffer audit events
	// internally, sending batch updates either once a certain number of
	// events have been received or a certain amount of time has passed.
	ModeBatch = "batch"
	// ModeBlocking causes the audit backend to block on every attempt to process
	// a set of events. This causes requests to the API server to wait for the
	// flush before sending a response.
	ModeBlocking = "blocking"
	// ModeBlockingStrict is the same as ModeBlocking, except when there is
	// a failure during audit logging at RequestReceived stage, the whole
	// request to apiserver will fail.
	ModeBlockingStrict = "blocking-strict"
)

// AllowedModes is the modes known for audit backends.
var AllowedModes = []string{
	ModeBatch,
	ModeBlocking,
	ModeBlockingStrict,
}

//type AuditOptions struct {
type config struct {
	// Policy configuration file for filtering audit events that are captured.
	// If unspecified, a default is provided.
	PolicyFile string `json:"policyFile" flag:"audit-policy-file" description:"Path to the file that defines the audit policy configuration."`

	// Plugin options
	LogOptions     AuditLogOptions     `json:"log"`
	WebhookOptions AuditWebhookOptions `json:"webhook"`
}

func (c *config) Validate() error {
	var allErrors []error
	allErrors = append(allErrors, c.LogOptions.Validate()...)
	allErrors = append(allErrors, c.WebhookOptions.Validate()...)

	return errors.NewAggregate(allErrors)
}

func (p *config) tags() map[string]*configer.FieldTag {
	tags := map[string]*configer.FieldTag{}
	for k, v := range p.LogOptions.tags() {
		tags["log."+k] = v
	}
	for k, v := range p.WebhookOptions.tags() {
		tags["webhook."+k] = v
	}

	return tags
}

type ignoreErrorsBackend struct {
	audit.Backend
}

func (i *ignoreErrorsBackend) ProcessEvents(ev ...*auditinternal.Event) bool {
	i.Backend.ProcessEvents(ev...)
	return true
}

func (i *ignoreErrorsBackend) String() string {
	return fmt.Sprintf("ignoreErrors<%s>", i.Backend)
}

type AuditBatchOptions struct {
	// Should the backend asynchronous batch events to the webhook backend or
	// should the backend block responses?
	//
	// Defaults to asynchronous batch events.
	Mode string `json:"mode"`
	// Configuration for batching backend. Only used in batch mode.
	pluginbuffered.BatchConfig
}

func flagf(format string, a ...interface{}) []string {
	return []string{fmt.Sprintf(format, a...)}
}

func (o *AuditBatchOptions) tags(pluginName string) map[string]*configer.FieldTag {
	return map[string]*configer.FieldTag{
		"mode":           {Flag: flagf("audit-%s-mode", pluginName), Description: "Strategy for sending audit events. Blocking indicates sending events should block server responses. Batch causes the backend to buffer and write events asynchronously. Known modes are " + strings.Join(AllowedModes, ",") + "."},
		"bufferSize":     {Flag: flagf("audit-%s-batch-buffer-size", pluginName)},
		"maxBatchSize":   {Flag: flagf("audit-%s-batch-max-size", pluginName)},
		"maxBatchWait":   {Flag: flagf("audit-%s-batch-max-wait", pluginName)},
		"throttleEnable": {Flag: flagf("audit-%s-batch-throttle-enable", pluginName)},
		"throttleQPS":    {Flag: flagf("audit-%s-batch-throttle-qps", pluginName)},
		"throttleBurst":  {Flag: flagf("audit-%s-batch-throttle-burst", pluginName)},
	}
}

func (o *AuditBatchOptions) wrapBackend(delegate audit.Backend) audit.Backend {
	if o.Mode == ModeBlockingStrict {
		return delegate
	}
	if o.Mode == ModeBlocking {
		return &ignoreErrorsBackend{Backend: delegate}
	}
	return pluginbuffered.NewBackend(delegate, o.BatchConfig)
}

type AuditTruncateOptions struct {
	// Whether truncating is enabled or not.
	Enabled bool `json:"enabled" description:"Whether event and batch truncating is enabled."`

	// Truncating configuration.
	plugintruncate.Config
}

func (o *AuditTruncateOptions) tags(pluginName string) map[string]*configer.FieldTag {
	return map[string]*configer.FieldTag{
		"enabled":      {Flag: flagf("audit-%s-truncate-enabled", pluginName)},
		"maxBatchSize": {Flag: flagf("audit-%s-truncate-max-batch-size", pluginName)},
		"maxEventSize": {Flag: flagf("audit-%s-truncate-max-event-size", pluginName)},
	}
}

func (o *AuditTruncateOptions) Validate(pluginName string) error {
	config := o.Config
	if config.MaxEventSize <= 0 {
		return fmt.Errorf("invalid audit truncate %s max event size %v, must be a positive number", pluginName, config.MaxEventSize)
	}
	if config.MaxBatchSize < config.MaxEventSize {
		return fmt.Errorf("invalid audit truncate %s max batch size %v, must be greater than "+
			"max event size (%v)", pluginName, config.MaxBatchSize, config.MaxEventSize)
	}
	return nil
}

func (o *AuditTruncateOptions) wrapBackend(delegate audit.Backend) audit.Backend {
	if !o.Enabled {
		return delegate
	}
	return plugintruncate.NewBackend(delegate, o.Config)
}

// AuditLogOptions determines the output of the structured audit log by default.
type AuditLogOptions struct {
	Path       string `json:"path" flag:"audit-log-path" description:"If set, all requests coming to the apiserver will be logged to this file.  '-' means standard out."`
	MaxAge     int    `json:"maxAge" flag:"audit-log-maxage" description:"The maximum number of days to retain old audit log files based on the timestamp encoded in their filename."`
	MaxBackups int    `json:"maxBackups" flag:"audit-log-maxbackup" description:"The maximum number of old audit log files to retain."`
	MaxSize    int    `json:"maxSize" flag:"audit-log-maxsize" description:"The maximum size in megabytes of the audit log file before it gets rotated."`
	Format     string `json:"format" flag:"audit-log-format" default:"json" description:"-"`
	Compress   bool   `json:"compress" flag:"audit-log-compress" description:"If set, the rotated log files will be compressed using gzip."`

	BatchOptions    AuditBatchOptions    `json:"batch"`
	TruncateOptions AuditTruncateOptions `json:"truncate"`

	// API group version used for serializing audit events.
	//GroupVersionString string
}

func (p *AuditLogOptions) tags() map[string]*configer.FieldTag {
	tags := map[string]*configer.FieldTag{
		"format": {Description: "Format of saved audits. \"legacy\" indicates 1-line text format for each event. \"json\" indicates structured json format. Known formats are " + strings.Join(pluginlog.AllowedFormats, ",") + "."},
	}
	for k, v := range p.BatchOptions.tags(pluginlog.PluginName) {
		tags["batch."+k] = v
	}
	for k, v := range p.TruncateOptions.tags(pluginlog.PluginName) {
		tags["truncate."+k] = v
	}

	return tags
}

func (o *AuditLogOptions) Validate() []error {
	// Check whether the log backend is enabled based on the options.
	if !o.enabled() {
		return nil
	}

	var allErrors []error

	if err := validateBackendBatchOptions(pluginlog.PluginName, o.BatchOptions); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := o.TruncateOptions.Validate(pluginlog.PluginName); err != nil {
		allErrors = append(allErrors, err)
	}

	// Check log format
	validFormat := false
	for _, f := range pluginlog.AllowedFormats {
		if f == o.Format {
			validFormat = true
			break
		}
	}
	if !validFormat {
		allErrors = append(allErrors, fmt.Errorf("invalid audit log format %s, allowed formats are %q", o.Format, strings.Join(pluginlog.AllowedFormats, ",")))
	}

	// Check validities of MaxAge, MaxBackups and MaxSize of log options, if file log backend is enabled.
	if o.MaxAge < 0 {
		allErrors = append(allErrors, fmt.Errorf("--audit-log-maxage %v can't be a negative number", o.MaxAge))
	}
	if o.MaxBackups < 0 {
		allErrors = append(allErrors, fmt.Errorf("--audit-log-maxbackup %v can't be a negative number", o.MaxBackups))
	}
	if o.MaxSize < 0 {
		allErrors = append(allErrors, fmt.Errorf("--audit-log-maxsize %v can't be a negative number", o.MaxSize))
	}

	return allErrors
}

// Check whether the log backend is enabled based on the options.
func (o *AuditLogOptions) enabled() bool {
	return o != nil && o.Path != ""
}

func (o *AuditLogOptions) getWriter() io.Writer {
	if !o.enabled() {
		return nil
	}

	var w io.Writer = os.Stdout
	if o.Path != "-" {
		w = &lumberjack.Logger{
			Filename:   o.Path,
			MaxAge:     o.MaxAge,
			MaxBackups: o.MaxBackups,
			MaxSize:    o.MaxSize,
			Compress:   o.Compress,
		}
	}
	return w
}

func (o *AuditLogOptions) newBackend(w io.Writer) audit.Backend {
	log := pluginlog.NewBackend(w, o.Format)
	log = o.BatchOptions.wrapBackend(log)
	log = o.TruncateOptions.wrapBackend(log)
	return log
}

func validateBackendMode(pluginName string, mode string) error {
	for _, m := range AllowedModes {
		if m == mode {
			return nil
		}
	}
	return fmt.Errorf("invalid audit %s mode %s, allowed modes are %q", pluginName, mode, strings.Join(AllowedModes, ","))
}

func validateBackendBatchOptions(pluginName string, options AuditBatchOptions) error {
	if err := validateBackendMode(pluginName, options.Mode); err != nil {
		return err
	}
	if options.Mode != ModeBatch {
		// Don't validate the unused options.
		return nil
	}
	config := options.BatchConfig
	if config.BufferSize <= 0 {
		return fmt.Errorf("invalid audit batch %s buffer size %v, must be a positive number", pluginName, config.BufferSize)
	}
	if config.MaxBatchSize <= 0 {
		return fmt.Errorf("invalid audit batch %s max batch size %v, must be a positive number", pluginName, config.MaxBatchSize)
	}
	if config.ThrottleEnable {
		if config.ThrottleQPS <= 0 {
			return fmt.Errorf("invalid audit batch %s throttle QPS %v, must be a positive number", pluginName, config.ThrottleQPS)
		}
		if config.ThrottleBurst <= 0 {
			return fmt.Errorf("invalid audit batch %s throttle burst %v, must be a positive number", pluginName, config.ThrottleBurst)
		}
	}
	return nil
}

// AuditWebhookOptions control the webhook configuration for audit events.
type AuditWebhookOptions struct {
	ConfigFile     string       `json:"configFile" flag:"audit-webhook-config-file" description:"Path to a kubeconfig formatted file that defines the audit webhook configuration."`
	InitialBackoff api.Duration `json:"initialBackoff" flag:"audit-webhook-initial-backoff" description:"The amount of time to wait before retrying the first failed request."`

	BatchOptions    AuditBatchOptions    `json:"batch"`
	TruncateOptions AuditTruncateOptions `json:"truncate"`

	// API group version used for serializing audit events.
	//GroupVersionString string
}

func (p *AuditWebhookOptions) tags() map[string]*configer.FieldTag {
	tags := map[string]*configer.FieldTag{}
	for k, v := range p.BatchOptions.tags(pluginwebhook.PluginName) {
		tags["batch."+k] = v
	}
	for k, v := range p.TruncateOptions.tags(pluginwebhook.PluginName) {
		tags["truncate."+k] = v
	}

	return tags
}

func (o *AuditWebhookOptions) Validate() []error {
	if !o.enabled() {
		return nil
	}

	var allErrors []error
	if err := validateBackendBatchOptions(pluginwebhook.PluginName, o.BatchOptions); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := o.TruncateOptions.Validate(pluginwebhook.PluginName); err != nil {
		allErrors = append(allErrors, err)
	}

	//if err := validateGroupVersionString(o.GroupVersionString); err != nil {
	//	allErrors = append(allErrors, err)
	//}
	return allErrors
}

func (o *AuditWebhookOptions) enabled() bool {
	return o != nil && o.ConfigFile != ""
}

// newUntruncatedBackend returns a webhook backend without the truncate options applied
// this is done so that the same trucate backend can wrap both the webhook and dynamic backends
func (o *AuditWebhookOptions) newUntruncatedBackend(customDial utilnet.DialFunc) (audit.Backend, error) {
	webhook, err := pluginwebhook.NewBackend(o.ConfigFile, webhook.DefaultRetryBackoffWithInitialDelay(o.InitialBackoff.Duration), customDial)
	if err != nil {
		return nil, fmt.Errorf("initializing audit webhook: %v", err)
	}
	webhook = o.BatchOptions.wrapBackend(webhook)
	return webhook, nil
}

// AuditDynamicOptions control the configuration of dynamic backends for audit events
type AuditDynamicOptions struct {
	// Enabled tells whether the dynamic audit capability is enabled.
	Enabled bool

	// Configuration for batching backend. This is currently only used as an override
	// for integration tests
	BatchConfig *pluginbuffered.BatchConfig
}

func newConfig() *config {
	return &config{
		WebhookOptions: AuditWebhookOptions{
			InitialBackoff: api.Duration{Duration: pluginwebhook.DefaultInitialBackoffDelay},
			BatchOptions: AuditBatchOptions{
				Mode:        ModeBatch,
				BatchConfig: defaultWebhookBatchConfig(),
			},
			TruncateOptions: NewAuditTruncateOptions(),
			//GroupVersionString: "audit.k8s.io/v1",
		},
		LogOptions: AuditLogOptions{
			Format: pluginlog.FormatJson,
			BatchOptions: AuditBatchOptions{
				Mode:        ModeBlocking,
				BatchConfig: defaultLogBatchConfig(),
			},
			TruncateOptions: NewAuditTruncateOptions(),
			//GroupVersionString: "audit.k8s.io/v1",
		},
	}
}

func NewAuditTruncateOptions() AuditTruncateOptions {
	return AuditTruncateOptions{
		Enabled: false,
		Config: plugintruncate.Config{
			MaxBatchSize: 10 * 1024 * 1024, // 10MB
			MaxEventSize: 100 * 1024,       // 100KB
		},
	}
}

// defaultWebhookBatchConfig returns the default BatchConfig used by the Webhook backend.
func defaultWebhookBatchConfig() pluginbuffered.BatchConfig {
	return pluginbuffered.BatchConfig{
		BufferSize:   defaultBatchBufferSize,
		MaxBatchSize: defaultBatchMaxSize,
		MaxBatchWait: api.Duration{Duration: defaultBatchMaxWait},

		ThrottleEnable: true,
		ThrottleQPS:    defaultBatchThrottleQPS,
		ThrottleBurst:  defaultBatchThrottleBurst,

		AsyncDelegate: true,
	}
}

// defaultLogBatchConfig returns the default BatchConfig used by the Log backend.
func defaultLogBatchConfig() pluginbuffered.BatchConfig {
	return pluginbuffered.BatchConfig{
		BufferSize: defaultBatchBufferSize,
		// Batching is not useful for the log-file backend.
		// MaxBatchWait ignored.
		MaxBatchSize:   1,
		ThrottleEnable: false,
		// Asynchronous log threads just create lock contention.
		AsyncDelegate: false,
	}
}

var (
	_module = &module{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUDIT,
	}, {
		Hook:        _module.stop,
		Owner:       moduleName,
		HookNum:     proc.ACTION_STOP,
		Priority:    proc.PRI_SYS_START,
		SubPriority: options.PRI_M_AUDIT,
	}}
)

type module struct {
	name    string
	config  *config
	ctx     context.Context
	cancel  context.CancelFunc
	checker policy.Checker
	backend audit.Backend
}

func (p *module) Checker() audit.Checker {
	return p.checker
}

func (p *module) Backend() audit.Backend {
	return p.backend
}

func (p *module) init(ctx context.Context) (err error) {
	c := proc.ConfigerMustFrom(ctx)
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := &config{}
	if err := c.Read(moduleName, cf); err != nil {
		return err
	}
	p.config = cf

	if err = p.initAudit(); err != nil {
		return err
	}

	if p.backend != nil {
		if err = p.backend.Run(p.ctx.Done()); err != nil {
			return err
		}
	}

	options.WithAudit(ctx, p)
	return
}

func (p *module) stop(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.backend != nil {
		p.backend.Shutdown()
	}

	return nil
}

func (p *module) initAudit() error {
	// 1. Build policy checker
	checker, err := p.config.newPolicyChecker()
	if err != nil {
		return err
	}

	c := p.config

	// 2. Build log backend
	var logBackend audit.Backend
	if w := c.LogOptions.getWriter(); w != nil {
		if checker == nil {
			klog.V(2).Info("No audit policy file provided, no events will be recorded for log backend")
		} else {
			logBackend = c.LogOptions.newBackend(w)
		}
	}

	// 3. Build webhook backend
	var webhookBackend audit.Backend
	if c.WebhookOptions.enabled() {
		if checker == nil {
			klog.V(2).Info("No audit policy file provided, no events will be recorded for webhook backend")
		} else {
			//if c.EgressSelector != nil {
			//	var egressDialer utilnet.DialFunc
			//	egressDialer, err = c.EgressSelector.Lookup(egressselector.ControlPlane.AsNetworkContext())
			//	if err != nil {
			//		return err
			//	}
			//	webhookBackend, err = o.WebhookOptions.newUntruncatedBackend(egressDialer)
			//} else {
			var d net.Dialer
			webhookBackend, err = c.WebhookOptions.newUntruncatedBackend(d.DialContext)
			//}
			if err != nil {
				return err
			}
		}
	}

	// 4. Apply dynamic options.
	var dynamicBackend audit.Backend
	if webhookBackend != nil {
		// if only webhook is enabled wrap it in the truncate options
		dynamicBackend = c.WebhookOptions.TruncateOptions.wrapBackend(webhookBackend)
	}

	// 5. Set the policy checker
	p.checker = checker

	// 6. Join the log backend with the webhooks
	p.backend = appendBackend(logBackend, dynamicBackend)

	if p.backend != nil {
		klog.V(2).Infof("Using audit backend: %s", p.backend)
	}

	return nil
}

func (c *config) newPolicyChecker() (policy.Checker, error) {
	if c.PolicyFile == "" {
		return nil, nil
	}

	if p, err := policy.LoadPolicyFromFile(c.PolicyFile); err != nil {
		return nil, fmt.Errorf("loading audit policy file: %v", err)
	} else {
		return policy.NewChecker(p), nil
	}
}

func RegisterHooks() {
	proc.RegisterHooks(hookOps)
}

func RegisterConfig() {
	cf := newConfig()
	proc.RegisterFlags(moduleName, "Audit", cf, configer.WithTags(cf.tags))
}

func Register() {
	RegisterHooks()
	RegisterConfig()
}
