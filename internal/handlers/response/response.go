package response

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"developer.dev/web-3/internal/model"
)

func Send(w http.ResponseWriter, status int, value any, err error) {
	if err != nil {
		Error(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func Error(w http.ResponseWriter, err error) {
	status, message, field := http.StatusInternalServerError, "Внутренняя ошибка сервера", ""
	var validation *model.ValidationError
	switch {
	case errors.As(err, &validation):
		status, message, field = http.StatusUnprocessableEntity, validation.Message, validation.Field
	case errors.Is(err, model.ErrNotFound):
		status, message = http.StatusNotFound, err.Error()
	case errors.Is(err, model.ErrUnauthorized), errors.Is(err, model.ErrCredentials):
		status, message = http.StatusUnauthorized, err.Error()
	case errors.Is(err, model.ErrForbidden):
		status, message = http.StatusForbidden, err.Error()
	case errors.Is(err, model.ErrEmailExists):
		status, message, field = http.StatusConflict, err.Error(), "email"
	case errors.Is(err, model.ErrInsufficientStock):
		status, message = http.StatusConflict, err.Error()
	default:
		slog.Error("request failed", "error", err)
	}
	Send(w, status, map[string]string{"error": message, "field": field}, nil)
}

func Decode(w http.ResponseWriter, r *http.Request, input any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &model.ValidationError{Field: "body", Message: "Используйте Content-Type: application/json"}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(input); err != nil {
		return &model.ValidationError{Field: "body", Message: "Проверьте формат JSON и названия полей"}
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return &model.ValidationError{Field: "body", Message: "Ожидается один JSON-объект"}
	}
	return nil
}

func ID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, &model.ValidationError{Field: "id", Message: "Некорректный идентификатор"}
	}
	return id, nil
}

func ProductFilter(r *http.Request) (model.ProductFilter, error) {
	query := r.URL.Query()
	filter := model.ProductFilter{Query: query.Get("q"), Sort: query.Get("sort")}
	if value := query.Get("category"); value != "" {
		categoryID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return filter, &model.ValidationError{Field: "category", Message: "Некорректная категория"}
		}
		filter.CategoryID = categoryID
	}
	return filter, nil
}
