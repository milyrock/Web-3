package create_feedback

import (
	"context"
	"net/http"

	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/model"
)

type Usecase interface {
	CreateFeedback(ctx context.Context, feedback model.Feedback, consent bool) (model.Feedback, error)
}

type Request struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Message string `json:"message"`
	Rating  *int   `json:"rating"`
	Consent bool   `json:"consent"`
}

func New(service Usecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input Request
		if err := response.Decode(w, r, &input); err != nil {
			response.Error(w, err)
			return
		}
		feedback, err := service.CreateFeedback(r.Context(), model.Feedback{
			Name: input.Name, Email: input.Email, Subject: input.Subject, Message: input.Message, Rating: input.Rating,
		}, input.Consent)
		response.Send(w, http.StatusCreated, response.Feedback(feedback), err)
	}
}
