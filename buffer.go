package psalm

type Buffer []float32

func (b Buffer) Clear() {
	for i := range b {
		b[i] = 0
	}
}
