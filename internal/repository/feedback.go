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

func (r *Repository) Feedback(ctx context.Context) ([]model.Feedback, error) {
	rows, err := sq.Select("id", "name", "email", "subject", "message", "rating", "status", "created_at").
		From("feedback").OrderBy("id DESC").PlaceholderFormat(sq.Dollar).RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.Feedback{}
	for rows.Next() {
		var feedback model.Feedback
		if err := rows.Scan(&feedback.ID, &feedback.Name, &feedback.Email, &feedback.Subject,
			&feedback.Message, &feedback.Rating, &feedback.Status, &feedback.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, feedback)
	}
	return items, rows.Err()
}
