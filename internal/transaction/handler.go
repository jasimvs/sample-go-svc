package transaction

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Transaction struct {
	ID        string  `json:"id"`
	Amount    float64 `json:"amount"`
	Type      string  `json:"type"`
	Timestamp string  `json:"timestamp"`
}

func GetHandler(c echo.Context) error {
	tx := Transaction{}

	return c.JSON(http.StatusOK, tx)
}

func PostHandler(c echo.Context) error {
	return c.JSON(http.StatusAccepted, nil)
}
