package audit

import (
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
	"github.com/SophearithSaing/synaptic-api/internal/web"
)

var decimalNumberPattern = regexp.MustCompile(
	`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`,
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
	query := request.URL.Query()
	errors := make([]string, 0)
	unknown := make([]string, 0)
	for name := range query {
		if name != "page" && name != "limit" {
			unknown = append(unknown, name)
		}
	}
	sort.Strings(unknown)
	for _, name := range unknown {
		errors = append(errors, "property "+name+" should not exist")
	}
	page, pageErrors := paginationValue(query, "page", defaultPage, 0)
	limit, limitErrors := paginationValue(query, "limit", defaultLimit, maxLimit)
	errors = append(errors, pageErrors...)
	errors = append(errors, limitErrors...)
	if len(errors) != 0 {
		return 0, 0, web.NewError(http.StatusBadRequest, errors)
	}
	if page-1 > math.MaxInt64/limit {
		return 0, 0, validationError("page", "must be within the supported range")
	}
	return page, limit, nil
}

// paginationValue parses one positive query integer with an optional maximum.
func paginationValue(query map[string][]string, name string, fallback int64, maximum int64) (int64, []string) {
	values, exists := query[name]
	if !exists {
		return fallback, nil
	}
	if len(values) != 1 {
		return 0, []string{name + " must be an integer number"}
	}
	value := ""
	value = values[0]
	trimmed := strings.TrimSpace(value)
	parsed, ok := parseJSDecimal(trimmed)
	if !ok || math.Trunc(parsed) != parsed || parsed >= float64(math.MaxInt64) || parsed < math.MinInt64 {
		return 0, []string{name + " must be an integer number"}
	}
	integer := int64(parsed)
	errors := make([]string, 0, 2)
	if integer < 1 {
		errors = append(errors, name+" must not be less than 1")
	}
	if maximum > 0 && integer > maximum {
		errors = append(errors, name+" must not be greater than "+strconv.FormatInt(maximum, 10))
	}
	return integer, errors
}

// parseJSDecimal accepts JavaScript decimal and exponent number syntax only.
func parseJSDecimal(value string) (float64, bool) {
	if value == "" {
		return 0, true
	}
	if !decimalNumberPattern.MatchString(value) {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	return parsed, err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
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
