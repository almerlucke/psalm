package psalm

const (
	Mono   = 1
	Stereo = 2
)

type Channels []Buffer

func NewChannels(n int, size int) Channels {
	p := make(Channels, n)
	for i, _ := range p {
		p[i] = make(Buffer, size)
	}
	return p
}

func (c Channels) Clear() {
	for i := range c {
		c[i].Clear()
	}
}

func (c Channels) IsStereo() bool {
	return len(c) == Stereo
}

func (c Channels) BufLen() int {
	return len(c[0])
}

func (c Channels) CopyWithFunc(c2 Channels, f func(Buffer, float32, int)) {
	if c.IsStereo() {
		if c2.IsStereo() {
			for i := 0; i < c.BufLen(); i++ {
				f(c[0], c2[0][i], i)
				f(c[1], c2[1][i], i)
			}
		} else {
			for i := 0; i < c.BufLen(); i++ {
				f(c[0], c2[0][i], i)
				f(c[1], c2[0][i], i)
			}
		}
	} else {
		if c2.IsStereo() {
			for i := 0; i < c.BufLen(); i++ {
				f(c[0], (c2[0][i]+c2[1][i])*0.5, i)
			}
		} else {
			for i := 0; i < c.BufLen(); i++ {
				f(c[0], c2[0][i], i)
			}
		}
	}
}

func (c Channels) Copy(c2 Channels) {
	if c.IsStereo() {
		if c2.IsStereo() {
			for i := 0; i < c.BufLen(); i++ {
				c[0][i] = c2[0][i]
				c[1][i] = c2[1][i]
			}
		} else {
			for i := 0; i < c.BufLen(); i++ {
				c[0][i] = c2[0][i]
				c[1][i] = c2[0][i]
			}
		}
	} else {
		if c2.IsStereo() {
			for i := 0; i < c.BufLen(); i++ {
				c[0][i] = (c2[0][i] + c2[1][i]) * 0.5
			}
		} else {
			for i := 0; i < c.BufLen(); i++ {
				c[0][i] = c2[0][i]
			}
		}
	}
}

func (c Channels) Accum(c2 Channels) {
	if c.IsStereo() {
		if c2.IsStereo() {
			for i := 0; i < c.BufLen(); i++ {
				c[0][i] += c2[0][i]
				c[1][i] += c2[1][i]
			}
		} else {
			for i := 0; i < c.BufLen(); i++ {
				c[0][i] += c2[0][i]
				c[1][i] += c2[0][i]
			}
		}
	} else {
		if c2.IsStereo() {
			for i := 0; i < c.BufLen(); i++ {
				c[0][i] += (c2[0][i] + c2[1][i]) * 0.5
			}
		} else {
			for i := 0; i < c.BufLen(); i++ {
				c[0][i] += c2[0][i]
			}
		}
	}
}
