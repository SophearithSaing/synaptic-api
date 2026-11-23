package catalog

import "context"

// Service implements catalog authoring workflows.
type Service struct {
	repo      AuthoringRepository
	validator *AuthoringValidator
}

// NewService builds a catalog authoring service.
func NewService(
	repo AuthoringRepository,
	validator *AuthoringValidator,
) *Service {
	return &Service{repo: repo, validator: validator}
}

// CreateCategory persists a validated category.
func (s *Service) CreateCategory(
	ctx context.Context,
	request CreateCategoryRequest,
) (*Category, []string, error) {
	if messages := s.validator.ValidateCreateCategory(request); len(messages) != 0 {
		return nil, messages, nil
	}
	category, err := s.repo.CreateCategory(ctx, request)
	return category, nil, err
}

// CreateTopic persists a validated topic.
func (s *Service) CreateTopic(
	ctx context.Context,
	request CreateTopicRequest,
) (*Topic, []string, error) {
	if messages := s.validator.ValidateCreateTopic(request); len(messages) != 0 {
		return nil, messages, nil
	}
	topic, err := s.repo.CreateTopic(ctx, request)
	return topic, nil, err
}

// CreateQuestionSets validates every request before creating each in input
// order. Storage errors leave any preceding creates intact.
func (s *Service) CreateQuestionSets(
	ctx context.Context,
	requests []CreateQuestionSetRequest,
) ([]QuestionSet, []string, error) {
	for _, request := range requests {
		if messages := s.validator.ValidateCreateQuestionSet(request); len(messages) != 0 {
			return nil, messages, nil
		}
	}
	created := make([]QuestionSet, 0, len(requests))
	for _, request := range requests {
		questionSet, err := s.repo.CreateQuestionSet(ctx, request)
		if err != nil {
			return nil, nil, err
		}
		created = append(created, *questionSet)
	}
	return created, nil, nil
}

// UpdateQuestionSet applies one validated patch.
func (s *Service) UpdateQuestionSet(
	ctx context.Context,
	id string,
	request UpdateQuestionSetRequest,
) (*QuestionSet, []string, error) {
	if messages := s.validator.ValidateUpdateQuestionSet(request); len(messages) != 0 {
		return nil, messages, nil
	}
	questionSet, err := s.repo.UpdateQuestionSet(ctx, id, request)
	return questionSet, nil, err
}

// UpdateQuestionSets validates all patches before attempting each one. It
// returns the first error in input order after every item has been attempted.
func (s *Service) UpdateQuestionSets(
	ctx context.Context,
	requests []BulkUpdateQuestionSetRequest,
) ([]QuestionSet, []string, error) {
	for _, request := range requests {
		if messages := s.validator.ValidateBulkUpdateQuestionSet(request); len(messages) != 0 {
			return nil, messages, nil
		}
	}
	updated := make([]QuestionSet, 0, len(requests))
	var first error
	for _, request := range requests {
		questionSet, err := s.repo.UpdateQuestionSet(
			ctx, request.ID, request.UpdateQuestionSetRequest,
		)
		if err != nil && first == nil {
			first = err
		}
		if err == nil {
			updated = append(updated, *questionSet)
		}
	}
	if first != nil {
		return nil, nil, first
	}
	return updated, nil, nil
}

// DeleteCategory deletes an unreferenced category.
func (s *Service) DeleteCategory(ctx context.Context, id string) error {
	return s.repo.DeleteCategory(ctx, id)
}

// DeleteTopic deletes an unreferenced topic.
func (s *Service) DeleteTopic(ctx context.Context, id string) error {
	return s.repo.DeleteTopic(ctx, id)
}

// DeleteQuestionSet deletes an unreferenced question set.
func (s *Service) DeleteQuestionSet(ctx context.Context, id string) error {
	return s.repo.DeleteQuestionSet(ctx, id)
}
