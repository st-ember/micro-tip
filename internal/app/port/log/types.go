package log

// Define a private underlying type to prevent collisions
type contextKey string

// TraceIDKey is the context key used for propagating request trace IDs
const TraceIDKey contextKey = "trace_id"
