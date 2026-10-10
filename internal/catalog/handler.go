package catalog

import (
	"context"
	"errors"
	"net/http"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// Handler owns the protected catalog read routes.
type Handler struct {
	repo          Repository
	service       *Service
	authenticator *identity.Authenticator
}

// NewHandler builds the catalog routes handler.
func NewHandler(
	repo Repository,
	authenticator *identity.Authenticator,
	services ...*Service,
) *Handler {
	var service *Service
	if len(services) != 0 {
		service = services[0]
	}
	return &Handler{repo: repo, service: service, authenticator: authenticator}
}

// Mount registers the catalog routes. Every route requires
// authentication.
func (h *Handler) Mount(mux *http.ServeMux) {
	auth := h.authenticator.Middleware

	mux.Handle("GET /categories/categories", auth(
		http.HandlerFunc(h.listCategories),
	))
	mux.Handle("GET /categories/{id}", auth(
		http.HandlerFunc(h.getCategory),
	))
	mux.Handle("GET /topics", auth(http.HandlerFunc(h.listTopics)))
	mux.Handle("GET /topics/{id}", auth(http.HandlerFunc(h.getTopic)))
	mux.Handle("GET /questions/{id}", auth(
		http.HandlerFunc(h.getQuestionSet),
	))
	mux.Handle("GET /questions/topic/{slug}", auth(
		http.HandlerFunc(h.listQuestionSetsByTopic),
	))
	if h.service == nil {
		return
	}
	admin := identity.RequireRole(identity.RoleAdmin)
	write := func(next http.HandlerFunc) http.Handler {
		return web.Chain(next, auth, admin, identity.RequireCSRF)
	}
	mux.Handle("POST /categories/category/create", write(h.createCategory))
	mux.Handle("POST /topics/create", write(h.createTopic))
	mux.Handle("POST /questions/create", write(h.createQuestionSets))
	mux.Handle("PATCH /questions/update", write(h.updateQuestionSets))
	mux.Handle("PATCH /questions/{id}", write(h.updateQuestionSet))
	mux.Handle("DELETE /categories/{id}", write(h.deleteCategory))
	mux.Handle("DELETE /topics/{id}", write(h.deleteTopic))
	mux.Handle("DELETE /questions/{id}", write(h.deleteQuestionSet))
}

// createCategory decodes and creates a category.
func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var request CreateCategoryRequest
	if !h.decode(w, r, &request) {
		return
	}
	created, messages, err := h.service.CreateCategory(r.Context(), request)
	h.writeAuthoring(w, r, http.StatusCreated, created, messages, err)
}

// createTopic decodes and creates a topic.
func (h *Handler) createTopic(w http.ResponseWriter, r *http.Request) {
	var request CreateTopicRequest
	if !h.decode(w, r, &request) {
		return
	}
	created, messages, err := h.service.CreateTopic(r.Context(), request)
	h.writeAuthoring(w, r, http.StatusCreated, created, messages, err)
}

// createQuestionSets decodes and creates question sets.
func (h *Handler) createQuestionSets(w http.ResponseWriter, r *http.Request) {
	var requests []CreateQuestionSetRequest
	if !h.decode(w, r, &requests) {
		return
	}
	created, messages, err := h.service.CreateQuestionSets(r.Context(), requests)
	h.writeAuthoring(w, r, http.StatusCreated, created, messages, err)
}

// updateQuestionSets decodes and applies bulk question-set patches.
func (h *Handler) updateQuestionSets(w http.ResponseWriter, r *http.Request) {
	var requests []BulkUpdateQuestionSetRequest
	if !h.decode(w, r, &requests) {
		return
	}
	updated, messages, err := h.service.UpdateQuestionSets(r.Context(), requests)
	h.writeAuthoring(w, r, http.StatusOK, updated, messages, err)
}

// updateQuestionSet decodes and applies one question-set patch.
func (h *Handler) updateQuestionSet(w http.ResponseWriter, r *http.Request) {
	var request UpdateQuestionSetRequest
	if !h.decode(w, r, &request) {
		return
	}
	updated, messages, err := h.service.UpdateQuestionSet(r.Context(), r.PathValue("id"), request)
	h.writeAuthoring(w, r, http.StatusOK, updated, messages, err)
}

// deleteCategory deletes one category.
func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	h.delete(w, r, h.service.DeleteCategory)
}

// deleteTopic deletes one topic.
func (h *Handler) deleteTopic(w http.ResponseWriter, r *http.Request) {
	h.delete(w, r, h.service.DeleteTopic)
}

// deleteQuestionSet deletes one question set.
func (h *Handler) deleteQuestionSet(w http.ResponseWriter, r *http.Request) {
	h.delete(w, r, h.service.DeleteQuestionSet)
}

// decode reads a strict JSON request or renders its pinned bad-request error.
func (h *Handler) decode(w http.ResponseWriter, r *http.Request, value any) bool {
	if err := web.DecodeJSON(w, r, value); err != nil {
		web.WriteError(w, r, err)
		return false
	}
	return true
}

// writeAuthoring renders validation, persistence, and successful results.
func (h *Handler) writeAuthoring(w http.ResponseWriter, r *http.Request, status int, value any, messages []string, err error) {
	if len(messages) != 0 {
		web.WriteError(w, r, web.NewError(http.StatusBadRequest, messages))
		return
	}
	if err != nil {
		writeLookupError(w, r, err)
		return
	}
	web.WriteJSON(w, status, value)
}

// delete runs one catalog deletion and writes an empty successful response.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request, remove func(context.Context, string) error) {
	if err := remove(r.Context(), r.PathValue("id")); err != nil {
		writeLookupError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// listCategories responds with all categories sorted by title.
func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repo.ListCategories(r.Context())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, categories)
}

// getCategory validates the id and returns one category.
func (h *Handler) getCategory(w http.ResponseWriter, r *http.Request) {
	category, err := h.repo.GetCategoryByID(r.Context(), r.PathValue("id"))
	if err != nil {
		writeLookupError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, category)
}

// listTopics responds with all topics sorted by title.
func (h *Handler) listTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := h.repo.ListTopics(r.Context())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, topics)
}

// getTopic validates the id and returns one topic.
func (h *Handler) getTopic(w http.ResponseWriter, r *http.Request) {
	topic, err := h.repo.GetTopicByID(r.Context(), r.PathValue("id"))
	if err != nil {
		writeLookupError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, topic)
}

// getQuestionSet returns one question set with its joined topic.
func (h *Handler) getQuestionSet(w http.ResponseWriter, r *http.Request) {
	questionSet, err := h.repo.GetQuestionSetByID(
		r.Context(), r.PathValue("id"),
	)
	if err != nil {
		writeLookupError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, questionSet)
}

// listQuestionSetsByTopic returns a topic's question sets.
func (h *Handler) listQuestionSetsByTopic(
	w http.ResponseWriter,
	r *http.Request,
) {
	questionSets, err := h.repo.ListQuestionSetsByTopicSlug(
		r.Context(), r.PathValue("slug"),
	)
	if err != nil {
		writeLookupError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, questionSets)
}

// writeLookupError maps catalog lookup errors to HTTP responses.
func writeLookupError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidObjectID):
		web.WriteError(w, r, web.NewError(
			http.StatusBadRequest, ErrInvalidObjectID.Error(),
		))
	case errors.Is(err, ErrCategoryNotFound):
		web.WriteError(w, r, web.NewError(
			http.StatusNotFound, ErrCategoryNotFound.Error(),
		))
	case errors.Is(err, ErrTopicNotFound):
		web.WriteError(w, r, web.NewError(
			http.StatusNotFound, ErrTopicNotFound.Error(),
		))
	case errors.Is(err, ErrQuestionSetNotFound):
		web.WriteError(w, r, web.NewError(
			http.StatusNotFound, ErrQuestionSetNotFound.Error(),
		))
	case errors.Is(err, ErrCategoryReferenced),
		errors.Is(err, ErrTopicReferenced),
		errors.Is(err, ErrQuestionSetReferenced):
		web.WriteError(w, r, web.NewError(http.StatusConflict, err.Error()))
	default:
		web.WriteError(w, r, err)
	}
}
