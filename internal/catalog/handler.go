package catalog

import (
	"errors"
	"net/http"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// Handler owns the protected catalog read routes.
type Handler struct {
	repo          Repository
	authenticator *identity.Authenticator
}

// NewHandler builds the catalog routes handler.
func NewHandler(
	repo Repository,
	authenticator *identity.Authenticator,
) *Handler {
	return &Handler{repo: repo, authenticator: authenticator}
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
	default:
		web.WriteError(w, r, err)
	}
}
