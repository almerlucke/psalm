package gen

import (
	"github.com/almerlucke/genny"
	"github.com/almerlucke/psalm"
)

type Handler[T any] func(genny.Generator[T], any, int)

type Config[T any] struct {
	Handler Handler[T]
}

type Gen[T any] struct {
	*psalm.Module
	gen     genny.Generator[T]
	handler Handler[T]
}

func NewWithConfig[T any](gen genny.Generator[T], config Config[T]) *Gen[T] {
	g := &Gen[T]{
		Module:  psalm.NewControl(),
		gen:     gen,
		handler: config.Handler,
	}

	g.SetSelf(g)

	return g
}

func New[T any](gen genny.Generator[T]) *Gen[T] {
	return NewWithConfig(gen, Config[T]{})
}

func (g *Gen[T]) Receive(value any, index int) {
	if psalm.IsBang(value) {
		switch index {
		case 0:
			v := g.gen.Generate()

			if g.gen.Done() {
				g.Send(psalm.Bang, 1)
			}

			g.Send(v, 0)
		case 1:
			g.gen.Reset()
		default:
			break
		}
	} else if g.handler != nil {
		g.handler(g.gen, value, index)
	}
}
