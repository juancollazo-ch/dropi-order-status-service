package compare

import (
	"errors"
	"log/slog"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/models" // Ajusta esta ruta según tu estructura real
)

// Result describe el resultado de la comparación
type Result struct {
	Changed      bool     // true si hubo cambio
	OldStatus    string   // status del penúltimo item
	NewStatus    string   // status del último item
	OrderID      int64    // id de la orden
	ProductNames []string // nombres de los productos en la orden
	HistorySize  int      // total de items en el history
}

// CompareOrderStatus evalúa si hubo cambio entre el último y penúltimo estado
func CompareOrderStatus(order *models.DropiOrder, logger *slog.Logger) (Result, error) {

	if order == nil {
		logger.Error("compare: order is nil")
		return Result{}, errors.New("order is nil")
	}

	hSize := len(order.History)

	// Necesitamos al menos 2 items para comparar
	if hSize < 2 {
		logger.Warn("compare: insufficient history length",
			"order_id", order.ID,
			"history_items", hSize,
		)
		return Result{}, errors.New("history must contain at least 2 records")
	}

	last := order.History[hSize-1]
	prev := order.History[hSize-2]

	logger.Info("compare: comparing history states",
		"order_id", order.ID,
		"previous_status", prev.Status,
		"last_status", last.Status,
	)

	changed := last.Status != prev.Status

	// Log informativo claro
	if changed {
		logger.Info("compare: status change detected",
			"order_id", order.ID,
			"from", prev.Status,
			"to", last.Status,
		)
	} else {
		logger.Info("compare: no change in status",
			"order_id", order.ID,
			"status", last.Status,
		)
	}

	return Result{
		Changed:      changed,
		OldStatus:    prev.Status,
		NewStatus:    last.Status,
		OrderID:      order.ID,
		ProductNames: order.GetProductNames(),
		HistorySize:  hSize,
	}, nil
}
