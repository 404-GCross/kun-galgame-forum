// Package namepref carries one reader's choice between a catalog record's
// Chinese name and its own (原名) across the request.
//
// It rides the context rather than a parameter because the choice has to reach
// every projection of the catalog face at once — roster rows, credits, voices,
// labels, series, search hits — and those render deep inside mappers that
// otherwise need nothing from the request.
package namepref

import "context"

type contextKey struct{}

func With(ctx context.Context, preferOriginal bool) context.Context {
	if !preferOriginal {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, true)
}

func PrefersOriginal(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	on, _ := ctx.Value(contextKey{}).(bool)
	return on
}
