package catalog

import (
	"net/http"
	"regexp"

	"github.com/SophearithSaing/synaptic-api/internal/identity"
	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// Handler owns the protected catalog read routes.
type Handler struct {
	repo          Repository
	authenticator *identity.Authenticator
}

// objectIDPattern mirrors the legacy MongoIdPipe validation.
var objectIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{24}$`)

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

	web.WriteJSON(w, http.StatusOK, orEmpty(categories))
}

// getCategory validates the id and returns one category.
func (h *Handler) getCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validObjectID(id) {
		writeInvalidObjectID(w, r)
		return
	}

	category, err := h.repo.CategoryByID(r.Context(), id)
	if notFound(w, r, err) {
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

	web.WriteJSON(w, http.StatusOK, orEmpty(topics))
}

// getTopic validates the id and returns one topic.
func (h *Handler) getTopic(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validObjectID(id) {
		writeInvalidObjectID(w, r)
		return
	}

	topic, err := h.repo.TopicByID(r.Context(), id)
	if notFound(w, r, err) {
		return
	}

	web.WriteJSON(w, http.StatusOK, topic)
}

// getQuestionSet returns one question set, populating the topic unless
// the populateTopic query value is exactly "false".
func (h *Handler) getQuestionSet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validObjectID(id) {
		writeInvalidObjectID(w, r)
		return
	}

	populate := r.URL.Query().Get("populateTopic") != "false"

	questionSet, err := h.repo.QuestionSetByID(r.Context(), id, populate)
	if notFound(w, r, err) {
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
	if notFound(w, r, err) {
		return
	}

	web.WriteJSON(w, http.StatusOK, orEmpty(questionSets))
}

// validObjectID mirrors the legacy MongoIdPipe validation.
func validObjectID(id string) bool {
	return objectIDPattern.MatchString(id)
}

// writeInvalidObjectID writes the pinned 400 body.
func writeInvalidObjectID(w http.ResponseWriter, r *http.Request) {
	web.WriteError(w, r, web.NewError(
		http.StatusBadRequest, "Invalid MongoDB ObjectId",
	))
}

// notFound writes the pinned 404 body for a missing document and
// reports whether it wrote anything. Other failures keep their error
// mapping.
func notFound(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(Error); !ok {
		web.WriteError(w, r, err)
		return true
	}

	web.WriteError(w, r, web.NewError(
		http.StatusNotFound, catalogMessage(err),
	))

	return true
}

// catalogMessage prefixes missing documents with their pinned bodies.
func catalogMessage(err error) string {
	if message, ok := err.(Error); ok {
		return string(message)
	}

	return err.Error()
}

// orEmpty keeps list responses as [] rather than null.
func orEmpty[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}
