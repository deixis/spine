package http

import (
	"context"
	"testing"

	lt "github.com/deixis/spine/testing"
)

func TestBuildMiddlewares(t *testing.T) {
	tt := lt.New(t)
	factory := &mwFactory{t: tt}
	ctx, _ := tt.WithCancel(context.Background())

	l := []Middleware{
		factory.newMiddleware(0),
		factory.newMiddleware(1),
		factory.newMiddleware(2),
	}
	e := &stdEndpoint{
		handleFunc: func(ctx context.Context, w ResponseWriter, r *Request) {},
	}

	c := buildMiddlewareChain(l, e)

	c(ctx, &responseWriter{}, &Request{})

	expected := 3
	if factory.C != expected {
		tt.Errorf("expect to be have %d middlewares called, but got %d", expected, factory.C)
	}
}

type mwFactory struct {
	N int
	C int
	t *lt.T
}

func (f *mwFactory) newMiddleware(expected int) Middleware {
	n := f.N
	f.N++

	return func(next ServeFunc) ServeFunc {
		return func(ctx context.Context, w ResponseWriter, r *Request) {
			f.C++
			if n != expected {
				f.t.Errorf("expect to be called in position %d, but got %d", expected, n)
			}
			next(ctx, w, r)
		}
	}
}
