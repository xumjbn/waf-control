package identity

import "context"

type ctxKey struct{}

func NewContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ctxKey{}, claims)
}

func GetClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(ctxKey{}).(*Claims)
	return claims
}
