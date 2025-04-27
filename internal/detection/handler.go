package detection

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetTransactions(c echo.Context) error {
	ctx := c.Request().Context()
	// Ideally, get the user ID from JWT token and pass it to the service to validate
	userID := c.QueryParam("user_id")
	if userID == "" {
		log.Println("Handler: Missing required query parameter 'user_id'")
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required query parameter: user_id")
	}

	suspiciousParam := c.QueryParam("suspicious")
	var isSuspiciousPtr *bool

	if suspiciousParam != "" {
		parsedBool, err := strconv.ParseBool(suspiciousParam)
		if err != nil {
			log.Printf("Handler: Invalid boolean value for 'suspicious' query parameter: %q, error: %v", suspiciousParam, err)
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid boolean value for query parameter 'suspicious': %s", suspiciousParam))
		}
		isSuspiciousPtr = &parsedBool
	}

	filter := Filter{
		UserID:       userID,
		IsSuspicious: isSuspiciousPtr,
	}

	txns, err := h.repo.Get(ctx, filter)
	if err != nil {
		log.Printf("Handler: Error calling repository Get: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve transactions")
	}

	log.Printf("Handler: Successfully retrieved %d transactions for user_id: %s", len(txns), userID)
	return c.JSON(http.StatusOK, txns)
}
