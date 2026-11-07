package domain

const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

const (
	ArticleComplete       = "complete"
	ArticleMetadataOnly   = "metadata_only"
	ArticleContentMissing = "content_missing"
	ArticleFailed         = "failed"
)

const (
	UserActive   = "active"
	UserDisabled = "disabled"
)

const (
	AgentOffline  = "offline"
	AgentOnline   = "online"
	AgentBusy     = "busy"
	AgentDisabled = "disabled"
	AgentError    = "error"
)

const (
	TaskPending   = "pending"
	TaskAssigned  = "assigned"
	TaskRunning   = "running"
	TaskSucceeded = "succeeded"
	TaskFailed    = "failed"
	TaskCanceled  = "canceled"
)

const (
	TaskTypeFetchArticles = "fetch_articles"
)

const (
	LoginPending    = "pending"
	LoginQRReady    = "qr_ready"
	LoginScanned    = "scanned"
	LoginConfirmed  = "confirmed"
	LoginExpired    = "expired"
	LoginFailed     = "failed"
	LoginCanceled   = "canceled"
	LoginSuperseded = "superseded"
)

var TerminalTaskStatuses = map[string]bool{
	TaskSucceeded: true,
	TaskFailed:    true,
	TaskCanceled:  true,
}

var ValidUserRoles = map[string]bool{
	RoleAdmin:    true,
	RoleOperator: true,
	RoleViewer:   true,
}
