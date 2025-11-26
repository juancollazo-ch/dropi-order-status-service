// internal/contextkeys/keys.go
package contextkeys

// contextKey es un tipo privado para evitar colisiones de claves en el contexto.
type contextKey string

const TraceIDKey contextKey = "trace_id"
const IDWorkspaceKey contextKey = "id_workspace"
const FlowNSKey contextKey = "flow_ns"
