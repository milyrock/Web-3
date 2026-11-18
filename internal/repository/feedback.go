package repository

import (
	"context"

	"developer.dev/web-3/internal/model"
	sq "github.com/Masterminds/squirrel"
)

func (r *Repository) CreateFeedback(ctx context.Context, feedback model.Feedback) (model.Feedback, error) {
	err := sq.Insert("feedback").Columns("name", "email", "subject", "message", "rating").
		Values(feedback.Name, feedback.Email, feedback.Subject, feedback.Message, feedback.Rating).
		Suffix("RETURNING id, status, created_at").PlaceholderFormat(sq.Dollar).RunWith(r.db).
		QueryRowContext(ctx).Scan(&feedback.ID, &feedback.Status, &feedback.CreatedAt)
	return feedback, err
}
