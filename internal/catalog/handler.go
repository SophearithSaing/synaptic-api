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
	categories, err := h.repo.Categories(r.Context())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, categories)
}

// getCategory validates the id and returns one category.
func (h *Handler) getCategory(w http.ResponseWriter, r *http.Request) {
	category, err := h.repo.CategoryByID(r.Context(), r.PathValue("id"))
	if err != nil {
		writeLookupError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, category)
}

// listTopics responds with all topics sorted by title.
func (h *Handler) listTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := h.repo.Topics(r.Context())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, topics)
}

// getTopic validates the id and returns one topic.
func (h *Handler) getTopic(w http.ResponseWriter, r *http.Request) {
	topic, err := h.repo.TopicByID(r.Context(), r.PathValue("id"))
	if err != nil {
		writeLookupError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, topic)
}

// getQuestionSet returns one question set, populating the topic unless
// the populateTopic query value is exactly "false".
func (h *Handler) getQuestionSet(w http.ResponseWriter, r *http.Request) {
	populate := r.URL.Query().Get("populateTopic") != "false"

	questionSet, err := h.repo.QuestionSetByID(
		r.Context(), r.PathValue("id"), populate,
	)
	if err != nil {
		writeLookupError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, questionSet)
}

// listQuestionSetsByTopic returns the topic's question sets, embedding
// the topic only when the populateTopic query value is exactly "true".
func (h *Handler) listQuestionSetsByTopic(
	w http.ResponseWriter,
	r *http.Request,
) {
	populate := r.URL.Query().Get("populateTopic") == "true"

	questionSets, err := h.repo.QuestionSetsByTopicSlug(
		r.Context(), r.PathValue("slug"), populate,
	)
	if err != nil {
		writeLookupError(w, r, err)
		return
	}

	web.WriteJSON(w, http.StatusOK, questionSets)
}

// writeLookupError maps catalog sentinels to their pinned bodies: an
// unparsable ObjectId is 400 and each missing document is 404 with
// its fixed message. Other failures keep their error mapping. Callers
// invoke it only for a non-nil error.
func writeLookupError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidObjectID):
		web.WriteError(w, r, web.NewError(
			http.StatusBadRequest, "Invalid MongoDB ObjectId",
		))
	case errors.Is(err, ErrCategoryNotFound):
		web.WriteError(w, r, web.NewError(
			http.StatusNotFound, "Category not found",
		))
	case errors.Is(err, ErrTopicNotFound):
		web.WriteError(w, r, web.NewError(
			http.StatusNotFound, "Topic not found",
		))
	case errors.Is(err, ErrQuestionSetNotFound):
		web.WriteError(w, r, web.NewError(
			http.StatusNotFound, "Question set not found",
		))
	default:
		web.WriteError(w, r, err)
	}
}
