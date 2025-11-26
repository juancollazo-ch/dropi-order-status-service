package compare

import (
	"errors"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/models"
	"go.uber.org/zap"
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
func CompareOrderStatus(order *models.DropiOrder) (Result, error) {

	if order == nil {
		zap.L().Error("compare: order is nil")
		return Result{}, errors.New("order is nil")
	}

	hSize := len(order.History)

	// Si no hay historial, no podemos procesar
	if hSize == 0 {
		zap.L().Warn("compare: no history records",
			zap.Int64("order_id", order.ID),
		)
		return Result{}, errors.New("history is empty")
	}

	last := order.History[hSize-1]

	// Si solo hay 1 registro, considerarlo como un cambio nuevo (primera vez que se registra)
	if hSize == 1 {
		zap.L().Info("compare: first history record (new order)",
			zap.Int64("order_id", order.ID),
			zap.String("status", last.Status),
		)
		return Result{
			Changed:      true, // Consideramos que es un cambio (nuevo)
			OldStatus:    "",   // No hay estado anterior
			NewStatus:    last.Status,
			OrderID:      order.ID,
			ProductNames: order.GetProductNames(),
			HistorySize:  hSize,
		}, nil
	}

	// Si hay 2 o más registros, comparar el último con el penúltimo
	prev := order.History[hSize-2]

	zap.L().Debug("compare: comparing history states",
		zap.Int64("order_id", order.ID),
		zap.String("previous_status", prev.Status),
		zap.String("last_status", last.Status),
	)

	changed := last.Status != prev.Status

	// Log informativo claro
	if changed {
		zap.L().Info("compare: status change detected",
			zap.Int64("order_id", order.ID),
			zap.String("from", prev.Status),
			zap.String("to", last.Status),
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
