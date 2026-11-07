package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"wechat-article-fetcher/server/internal/domain"
)

func (a *API) recordAudit(
	r *http.Request,
	action, resourceType, resourceID string,
	details any,
) {
	principal, ok := currentPrincipal(r.Context())
	if !ok {
		return
	}
	var encoded json.RawMessage
	if details != nil {
		if value, err := json.Marshal(details); err == nil {
			encoded = value
		}
	}
	actorID := principal.User.ID
	_ = a.store.AddAuditLog(r.Context(), domain.AuditLog{
		ActorUserID:  &actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      encoded,
		CreatedAt:    time.Now().UTC(),
	})
}
