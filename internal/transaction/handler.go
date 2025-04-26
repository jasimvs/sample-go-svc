package transaction

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/jasimvs/sample-go-svc/internal/model"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) CreateTransaction(c echo.Context) error {
	// Ideally, get the user ID from JWT token and pass it to the service to validate
	var req model.Transaction
	if err := c.Bind(&req); err != nil {
		log.Printf("Handler: Error binding request for create transaction: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
	}

	log.Printf("Handler: Received POST request to create transaction: %+v (ID may be empty)", req)

	createdTx, err := h.service.CreateTransaction(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			log.Printf("Handler: Validation error from service: %v", err)

			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, ErrConflict) {
			log.Printf("Handler: Conflict error from service: %v", err)

			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}

		log.Printf("Handler: Internal error from service for CreateTransaction: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create transaction")
	}

	log.Printf("Handler: Successfully created transaction ID %s, returning response.", createdTx.ID)
	return c.JSON(http.StatusCreated, createdTx)
}
