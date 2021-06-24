package rest

const (
	MODULE_SSO_BASE = 1 << 6 << iota
	MODULE_GATHER_BASE
)

// for req context
const (
	RshDataKey   = "rshData"
	RshConnKey   = "rshConn"
	ReqEntityKey = "reqEntity"
)

const (
	_               = 8000 + 10*iota
	MODULE_SSO_PORT // 8010
)

const (

	// action
	ActionInstall  = "install"
	ActionUpgrade  = "upgrade"
	ActionDelete   = "delete"
	ActionRollback = "rollback"
	ActionInit     = "init"
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionCopy     = "copy"
	ActionMove     = "move"
	ActionStart    = "start"
	ActionStop     = "stop"
	ActionRsh      = "rsh"
	ActionLogin    = "login"
	ActionExec     = "exec"
	ActionRun      = "run"
	ActionApply    = "apply"
	ActionRestore  = "restore"
	ActionReboot   = "reboot"
)
