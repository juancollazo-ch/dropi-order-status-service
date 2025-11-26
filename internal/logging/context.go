// internal/logging/context.go
package logging

import (
	"context"

	"go.uber.org/zap"
)

// contextKey es un tipo privado para evitar colisiones de claves de contexto.
type contextKey string

const IDWorkspaceKey contextKey = "id_workspace"
const FlowNSKey contextKey = "flow_ns"

// GetLoggingFieldsFromContext extrae los campos de logging (id_workspace, flow_ns)
// del contexto y los devuelve como un slice de zap.Field.
func GetLoggingFieldsFromContext(ctx context.Context) []zap.Field {
	fields := []zap.Field{}
	if idw, ok := ctx.Value(IDWorkspaceKey).(string); ok && idw != "" {
		fields = append(fields, zap.String("id_workspace", idw))
	}
	if fns, ok := ctx.Value(FlowNSKey).(string); ok && fns != "" {
		fields = append(fields, zap.String("flow_ns", fns))
	}
	return fields
}

// WithLoggingFields añade id_workspace y flow_ns al contexto si están presentes.
func WithLoggingFields(ctx context.Context, idWorkspace, flowNS string) context.Context {
	if idWorkspace != "" {
		ctx = context.WithValue(ctx, IDWorkspaceKey, idWorkspace)
	}
	if flowNS != "" {
		ctx = context.WithValue(ctx, FlowNSKey, flowNS)
	}
	return ctx
}
