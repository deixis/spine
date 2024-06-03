package bg

import (
	"context"

	scontext "github.com/deixis/spine/context"
)

func BG(parent context.Context, f func(ctx context.Context)) error {
	tr := scontext.TransitFromContext(parent)

	return RegFromContext(parent).Dispatch(NewTask(func() {
		ctx, cancel := context.WithCancel(context.WithoutCancel(parent))
		defer cancel()

		if tr != nil {
			ctx = scontext.TransitWithContext(ctx, tr)
		} else {
			ctx, tr = scontext.NewTransitWithContext(ctx)
		}
		scontext.ShipmentRange(parent, func(k string, v interface{}) bool {
			ctx = scontext.WithShipment(ctx, k, v)
			return true
		})

		f(ctx)
	}))
}

// Dispatch calls `Dispatch` on the context `Registry`
func Dispatch(ctx context.Context, j Job) error {
	return RegFromContext(ctx).Dispatch(j)
}

type contextKey struct{}

var activeContextKey = contextKey{}

// RegFromContext returns a `Reg` instance associated with `ctx`, or
// a new `Reg` if no existing `Reg` instance could be found.
func RegFromContext(ctx context.Context) *Reg {
	val := ctx.Value(activeContextKey)
	if o, ok := val.(*Reg); ok {
		return o
	}
	return NewReg("unnamed", ctx)
}

// RegWithContext returns a copy of parent in which `Reg` is stored
func RegWithContext(ctx context.Context, r *Reg) context.Context {
	return context.WithValue(ctx, activeContextKey, r)
}
