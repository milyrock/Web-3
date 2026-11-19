package usecase

import (
	"context"
	"net/mail"
	"strings"

	"developer.dev/web-3/internal/model"
)

func (s *Service) CreateFeedback(ctx context.Context, feedback model.Feedback, consent bool) (model.Feedback, error) {
	feedback.Name = strings.TrimSpace(feedback.Name)
	feedback.Email = strings.ToLower(strings.TrimSpace(feedback.Email))
	feedback.Subject, feedback.Message = strings.TrimSpace(feedback.Subject), strings.TrimSpace(feedback.Message)
	if !length(feedback.Name, 2, 100) {
		return model.Feedback{}, invalid("name", "Имя должно содержать от 2 до 100 символов")
	}
	address, err := mail.ParseAddress(feedback.Email)
	if err != nil || address.Address != feedback.Email || len(feedback.Email) > 254 {
		return model.Feedback{}, invalid("email", "Укажите корректный email")
	}
	if !length(feedback.Subject, 3, 150) {
		return model.Feedback{}, invalid("subject", "Тема должна содержать от 3 до 150 символов")
	}
	if !length(feedback.Message, 10, 5000) {
		return model.Feedback{}, invalid("message", "Сообщение должно содержать от 10 до 5000 символов")
	}
	if feedback.Rating != nil && (*feedback.Rating < 1 || *feedback.Rating > 5) {
		return model.Feedback{}, invalid("rating", "Оценка должна быть от 1 до 5")
	}
	if !consent {
		return model.Feedback{}, invalid("consent", "Нужно согласие на обработку имени и email")
	}
	return s.repo.CreateFeedback(ctx, feedback)
}

func (s *Service) Feedback(ctx context.Context, token string) ([]model.Feedback, error) {
	user, err := s.CurrentUser(ctx, token)
	if err != nil {
		return nil, err
	}
	if user.Role != model.RoleAdmin {
		return nil, model.ErrForbidden
	}
	return s.repo.Feedback(ctx)
}
