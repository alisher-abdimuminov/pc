package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"pc/internal/apperr"
)

type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func NewErrorHandler(log *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var appErr *apperr.Error
		var fiberErr *fiber.Error
		switch {
		case errors.As(err, &appErr):
		case errors.As(err, &fiberErr):
			appErr = &apperr.Error{Status: fiberErr.Code, Code: codeForStatus(fiberErr.Code), Message: messageForStatus(fiberErr.Code)}
		default:
			log.Error("unhandled error", zap.Error(err), zap.String("path", c.Path()))
			appErr = &apperr.Error{Status: fiber.StatusInternalServerError, Code: "internal_error", Message: messageForStatus(500)}
		}
		return c.Status(appErr.Status).JSON(fiber.Map{"error": ErrorBody{Code: appErr.Code, Message: appErr.Message, Fields: appErr.Fields}})
	}
}

func codeForStatus(s int) string {
	switch s {
	case 400:
		return "bad_request"
	case 401:
		return "unauthorized"
	case 403:
		return "forbidden"
	case 404:
		return "not_found"
	case 429:
		return "too_many_requests"
	default:
		return "internal_error"
	}
}

func messageForStatus(s int) string {
	switch s {
	case 400:
		return "Noto'g'ri so'rov"
	case 401:
		return "Avtorizatsiyadan o'tilmagan"
	case 403:
		return "Ruxsat yo'q"
	case 404:
		return "Topilmadi"
	case 429:
		return "Juda ko'p so'rov, keyinroq urinib ko'ring"
	default:
		return "Ichki xatolik"
	}
}
