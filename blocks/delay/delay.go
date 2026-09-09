package delay

// Delay structure
type Delay struct {
	Buffer   []float64
	WriteLoc int
	Length   int
}

// New create a new delay
func New(length int) *Delay {
	return &Delay{
		Buffer: make([]float64, length),
		Length: length,
	}
}

// Write to delay
func (d *Delay) Write(in float64) {
	d.Buffer[d.WriteLoc] = in
	d.WriteLoc = (d.WriteLoc - 1 + d.Length) % d.Length
}

func (d *Delay) ReadLinear(loc float64) float64 {
	iloc := int(loc)
	v := d.Buffer[(d.WriteLoc+iloc)%d.Length]
	return v + (d.Buffer[(d.WriteLoc+iloc+1)%d.Length]-v)*(loc-float64(iloc))
}

func (d *Delay) ReadHermite(loc float64) float64 {
	iloc := int(loc)
	f := loc - float64(iloc)
	t := d.WriteLoc + iloc + d.Length
	xm1 := d.Buffer[(t-1)%d.Length]
	x0 := d.Buffer[t%d.Length]
	x1 := d.Buffer[(t+1)%d.Length]
	x2 := d.Buffer[(t+2)%d.Length]
	c := (x1 - xm1) * 0.5
	v := x0 - x1
	w := c + v
	a := w + v + (x2-x0)*0.5
	bNeg := w + a

	return (((a*f)-bNeg)*f+c)*f + x0
}
