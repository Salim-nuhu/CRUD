package auth

import "context"

// Identity is who the request is from, as proven by a valid token.
type Identity struct {
	UserID string
	Role   string
}

// ctxKey is a private type, so no other package can collide with our key.
type ctxKey struct{}

// WithIdentity returns a copy of ctx that carries the logged-in identity.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// IdentityFromContext returns the logged-in identity, if the request has one.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok && id.UserID != ""
}

// UserIDFromContext is kept so the code we already wrote keeps working.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := IdentityFromContext(ctx)
	return id.UserID, ok
}
