package audit

import (
	"net/http"
	"strconv"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
	"github.com/SophearithSaing/synaptic-api/internal/web"
)

const (
	defaultPage  = int64(1)
	defaultLimit = int64(20)
	maxLimit     = int64(100)
)

// Handler owns the protected AI audit-log route.
type Handler struct {
	repository    Repository
	authenticator *identity.Authenticator
}

// NewHandler builds the AI audit-log handler.
func NewHandler(repository Repository, authenticator *identity.Authenticator) *Handler {
	return &Handler{repository: repository, authenticator: authenticator}
}

// Mount registers the administrator-only audit-log route.
func (handler *Handler) Mount(mux *http.ServeMux) {
	auth := handler.authenticator.Middleware
	admin := identity.RequireRole(identity.RoleAdmin)
	mux.Handle("GET /ai/logs", auth(admin(http.HandlerFunc(handler.list))))
}

// list validates pagination and renders one audit-log page.
func (handler *Handler) list(writer http.ResponseWriter, request *http.Request) {
	page, limit, err := pagination(request)
	if err != nil {
		web.WriteError(writer, request, err)
		return
	}
	result, err := handler.repository.List(request.Context(), page, limit)
	if err != nil {
		web.WriteError(writer, request, err)
		return
	}
	web.WriteJSON(writer, http.StatusOK, pageResponse{Items: result.Items,
		Total: result.Total, Page: result.Page, Limit: result.Limit})
}

// pagination reads legacy one-based page and limit query values.
func pagination(request *http.Request) (int64, int64, error) {
	page, err := paginationValue(request, "page", defaultPage, 0)
	if err != nil {
		return 0, 0, err
	}
	limit, err := paginationValue(request, "limit", defaultLimit, maxLimit)
	if err != nil {
		return 0, 0, err
	}
	return page, limit, nil
}

// paginationValue parses one positive query integer with an optional maximum.
func paginationValue(request *http.Request, name string, fallback int64, maximum int64) (int64, error) {
	value := request.URL.Query().Get(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, validationError(name, "must be an integer")
	}
	if parsed < 1 {
		return 0, validationError(name, "must not be less than 1")
	}
	if maximum > 0 && parsed > maximum {
		return 0, validationError(name, "must not be greater than "+strconv.FormatInt(maximum, 10))
	}
	return parsed, nil
}

// validationError produces the pinned legacy validation body.
func validationError(name string, message string) error {
	return web.NewError(http.StatusBadRequest, []string{name + " " + message})
}

type pageResponse struct {
	Items []Record `json:"items"`
	Total int64    `json:"total"`
	Page  int64    `json:"page"`
	Limit int64    `json:"limit"`
}
