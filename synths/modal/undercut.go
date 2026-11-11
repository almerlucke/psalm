package modal

import (
	"math"
	"sort"
)

// The undercut bar. A marimba, vibraphone or xylophone bar is not a uniform beam: its underside is
// arched away in the middle so the overtones fall on musical ratios (1 : 4 : 10 for a marimba). Here
// the bar is an Euler-Bernoulli beam, free at both ends, whose thickness h(x) varies along it: the
// stiffness goes as h^3 and the mass as h. It is discretised with 40 cubic Hermite finite elements
// (82 degrees of freedom) and the lowest modes of K w = omega^2 M w are found by subspace (inverse)
// iteration with a Rayleigh-Ritz step. The depth and width of the arch are then tuned (Newton) so the
// second and third modes sit on the wanted ratios; the solution is kept for a few depths between a
// uniform bar (undercut 0) and the tuned arch (undercut 1), and the strike reads the mode shapes
// from it.

const (
	beamElems   = 40
	beamNodes   = beamElems + 1
	beamDof     = 2 * beamNodes
	archModes   = 6 // modes kept (the rigid-body modes not counted)
	archSteps   = 8 // the undercut grid: 0, 1/8 ... 1
	archSubDim  = archModes + 2 + 4
	archIterMax = 60
)

// beamSolution is the lowest modes of one bar profile
type beamSolution struct {
	ratio [archModes]float64            // frequency over the first mode's
	w     [archModes][beamNodes]float64 // displacement at the nodes, mean square 1, positive at the left end
	th    [archModes][beamNodes]float64 // slope at the nodes, the same scaling
}

// at is the displacement of mode k at x (0 to 1), by Hermite interpolation
func (s *beamSolution) at(k int, x float64) float64 {
	x = max(0, min(x, 1))
	e := int(x * beamElems)
	if e >= beamElems {
		e = beamElems - 1
	}
	le := 1.0 / beamElems
	xi := (x - float64(e)*le) / le
	n1 := 1 - 3*xi*xi + 2*xi*xi*xi
	n2 := le * (xi - 2*xi*xi + xi*xi*xi)
	n3 := 3*xi*xi - 2*xi*xi*xi
	n4 := le * (-xi*xi + xi*xi*xi)
	return n1*s.w[k][e] + n2*s.th[k][e] + n3*s.w[k][e+1] + n4*s.th[k][e+1]
}

// gauss3 are the nodes and weights of 3-point Gauss quadrature on 0 to 1
var gauss3 = [3][2]float64{
	{0.5 - 0.5*math.Sqrt(0.6), 5.0 / 18},
	{0.5, 8.0 / 18},
	{0.5 + 0.5*math.Sqrt(0.6), 5.0 / 18},
}

// assembleBeam builds the stiffness and mass of a free-free beam of thickness h(x), unit length
func assembleBeam(h func(float64) float64) (k, m [][]float64) {
	k, m = square(beamDof), square(beamDof)
	le := 1.0 / beamElems
	for e := range beamElems {
		var ke, me [4][4]float64
		for _, g := range gauss3 {
			xi, wg := g[0], g[1]
			th := h((float64(e) + xi) * le)
			ei, mass := th*th*th, th
			n := [4]float64{1 - 3*xi*xi + 2*xi*xi*xi, le * (xi - 2*xi*xi + xi*xi*xi), 3*xi*xi - 2*xi*xi*xi, le * (-xi*xi + xi*xi*xi)}
			b := [4]float64{(-6 + 12*xi) / (le * le), (-4 + 6*xi) / le, (6 - 12*xi) / (le * le), (-2 + 6*xi) / le}
			for a := range 4 {
				for c := range 4 {
					ke[a][c] += wg * le * ei * b[a] * b[c]
					me[a][c] += wg * le * mass * n[a] * n[c]
				}
			}
		}
		for a := range 4 {
			for c := range 4 {
				k[2*e+a][2*e+c] += ke[a][c]
				m[2*e+a][2*e+c] += me[a][c]
			}
		}
	}
	return k, m
}

func square(n int) [][]float64 {
	a := make([][]float64, n)
	for i := range a {
		a[i] = make([]float64, n)
	}
	return a
}

// cholesky factors a symmetric positive definite matrix in place into its lower triangle
func cholesky(a [][]float64) {
	n := len(a)
	for j := range n {
		s := a[j][j]
		for k := range j {
			s -= a[j][k] * a[j][k]
		}
		a[j][j] = math.Sqrt(s)
		for i := j + 1; i < n; i++ {
			s := a[i][j]
			for k := range j {
				s -= a[i][k] * a[j][k]
			}
			a[i][j] = s / a[j][j]
		}
	}
}

// cholSolve solves L L^T x = b in place
func cholSolve(l [][]float64, b []float64) {
	n := len(l)
	for i := range n {
		s := b[i]
		for k := range i {
			s -= l[i][k] * b[k]
		}
		b[i] = s / l[i][i]
	}
	for i := n - 1; i >= 0; i-- {
		s := b[i]
		for k := i + 1; k < n; k++ {
			s -= l[k][i] * b[k]
		}
		b[i] = s / l[i][i]
	}
}

// jacobi finds the eigenvalues and (column) eigenvectors of a small symmetric matrix
func jacobi(a [][]float64) ([]float64, [][]float64) {
	n := len(a)
	v := square(n)
	for i := range n {
		v[i][i] = 1
	}
	for range 60 {
		off := 0.0
		for i := range n {
			for j := i + 1; j < n; j++ {
				off += a[i][j] * a[i][j]
			}
		}
		if off < 1e-30 {
			break
		}
		for p := range n {
			for q := p + 1; q < n; q++ {
				if math.Abs(a[p][q]) < 1e-300 {
					continue
				}
				th := (a[q][q] - a[p][p]) / (2 * a[p][q])
				t := 1 / (math.Abs(th) + math.Sqrt(th*th+1))
				if th < 0 {
					t = -t
				}
				c := 1 / math.Sqrt(t*t+1)
				s := t * c
				for k := range n {
					akp, akq := a[k][p], a[k][q]
					a[k][p], a[k][q] = c*akp-s*akq, s*akp+c*akq
				}
				for k := range n {
					apk, aqk := a[p][k], a[q][k]
					a[p][k], a[q][k] = c*apk-s*aqk, s*apk+c*aqk
				}
				for k := range n {
					vkp, vkq := v[k][p], v[k][q]
					v[k][p], v[k][q] = c*vkp-s*vkq, s*vkp+c*vkq
				}
			}
		}
	}
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = a[i][i]
	}
	return vals, v
}

func matVec(a [][]float64, x []float64, out []float64) {
	for i := range a {
		s := 0.0
		for j, aij := range a[i] {
			s += aij * x[j]
		}
		out[i] = s
	}
}

// subspace is the block of vectors the iteration keeps, so a nearby profile starts from its neighbour
type subspace [][]float64

func newSubspace() subspace {
	x := make(subspace, archSubDim)
	seed := uint32(12345)
	for j := range x {
		x[j] = make([]float64, beamDof)
		for i := range x[j] {
			seed = seed*1664525 + 1013904223
			x[j][i] = float64(seed>>8)/float64(1<<24) - 0.5
		}
	}
	return x
}

// solveBeam finds the lowest modes of a bar of thickness h(x); x0 is the start of the iteration and
// holds the answer afterwards
func solveBeam(h func(float64) float64, x0 subspace) *beamSolution {
	K, M := assembleBeam(h)
	const alpha = 200.0 // the shift: K + alpha M is positive definite, the rigid-body modes are not singular
	A := square(beamDof)
	for i := range A {
		for j := range A[i] {
			A[i][j] = K[i][j] + alpha*M[i][j]
		}
	}
	cholesky(A)
	q := archSubDim
	y := x0
	tmp := make([]float64, beamDof)
	var theta []float64
	var z [][]float64
	var lm [][]float64
	prev := make([]float64, q)
	for it := range archIterMax {
		for j := range q {
			matVec(M, y[j], tmp)
			copy(y[j], tmp)
			cholSolve(A, y[j])
		}
		// Rayleigh-Ritz on the block
		kr, mr := square(q), square(q)
		ky, my := make([][]float64, q), make([][]float64, q)
		for j := range q {
			ky[j], my[j] = make([]float64, beamDof), make([]float64, beamDof)
			matVec(K, y[j], ky[j])
			matVec(M, y[j], my[j])
		}
		for i := range q {
			for j := range q {
				var a, b float64
				for r := range beamDof {
					a += y[i][r] * ky[j][r]
					b += y[i][r] * my[j][r]
				}
				kr[i][j], mr[i][j] = a, b
			}
		}
		cholesky(mr)
		lm = mr
		// C = L^-1 Kr L^-T
		c := square(q)
		for j := range q { // columns of Kr through L^-1
			col := make([]float64, q)
			for i := range q {
				col[i] = kr[i][j]
			}
			for i := range q {
				s := col[i]
				for k := range i {
					s -= lm[i][k] * col[k]
				}
				col[i] = s / lm[i][i]
			}
			for i := range q {
				c[i][j] = col[i]
			}
		}
		for i := range q { // rows through L^-T
			row := c[i]
			for j := range q {
				s := row[j]
				for k := range j {
					s -= lm[j][k] * row[k]
				}
				row[j] = s / lm[j][j]
			}
		}
		for i := range q {
			for j := i + 1; j < q; j++ {
				c[i][j] = (c[i][j] + c[j][i]) / 2
				c[j][i] = c[i][j]
			}
		}
		theta, z = jacobi(c)
		order := make([]int, q)
		for i := range order {
			order[i] = i
		}
		sort.Slice(order, func(a, b int) bool { return theta[order[a]] < theta[order[b]] })
		// x = Y L^-T z for the sorted eigenvectors
		next := make(subspace, q)
		sortedTheta := make([]float64, q)
		for jj, oj := range order {
			sortedTheta[jj] = theta[oj]
			coef := make([]float64, q)
			for i := range q {
				coef[i] = z[i][oj]
			}
			for i := q - 1; i >= 0; i-- { // back substitution with L^T
				s := coef[i]
				for k := i + 1; k < q; k++ {
					s -= lm[k][i] * coef[k]
				}
				coef[i] = s / lm[i][i]
			}
			next[jj] = make([]float64, beamDof)
			for i := range q {
				for r := range beamDof {
					next[jj][r] += coef[i] * y[i][r]
				}
			}
		}
		theta = sortedTheta
		y = next
		done := it > 2
		for j := 2; j < 2+archModes; j++ {
			if math.Abs(theta[j]-prev[j]) > 1e-9*theta[j] {
				done = false
			}
		}
		copy(prev, theta)
		if done {
			break
		}
	}
	copy(x0, y)
	sol := &beamSolution{}
	for k := range archModes {
		vec := y[2+k]
		sol.ratio[k] = math.Sqrt(theta[2+k] / theta[2])
		// the mean square of the displacement, sampled finely
		for i := range beamNodes {
			sol.w[k][i], sol.th[k][i] = vec[2*i], vec[2*i+1]
		}
		var e float64
		const per = 6
		for i := range beamElems * per {
			v := sol.at(k, (float64(i)+0.5)/float64(beamElems*per))
			e += v * v
		}
		scale := 1 / math.Sqrt(e/float64(beamElems*per))
		if sol.w[k][0] < 0 {
			scale = -scale
		}
		for i := range beamNodes {
			sol.w[k][i] *= scale
			sol.th[k][i] *= scale
		}
	}
	return sol
}

// archProfile is the thickness of a bar arched away in the middle: depth 0-1 of the thickness at its
// deepest, the arch spanning width (as a fraction of the bar) with a smooth (1 - xi^2)^e shape
func archProfile(depth, width, e float64) func(float64) float64 {
	return func(x float64) float64 {
		xi := (x - 0.5) / width
		if math.Abs(xi) >= 1 {
			return 1
		}
		return 1 - depth*math.Pow(1-xi*xi, e)
	}
}

// UndercutRatios solves a bar of the given arch (depth 0 is a uniform bar) and returns the frequency
// ratios of its first modes
func UndercutRatios(depth, width, e float64) []float64 {
	s := solveBeam(archProfile(depth, width, e), newSubspace())
	return s.ratio[:]
}

// tuneArch finds the depth and width of the arch that put the second and third modes at the ratios
// r2 and r3 (Newton on the logarithms, finite-difference Jacobian)
func tuneArch(r2, r3, e float64, depth, width float64) (float64, float64, *beamSolution) {
	sub := newSubspace()
	eval := func(d, w float64) (float64, float64, *beamSolution) {
		s := solveBeam(archProfile(d, w, e), sub)
		return math.Log(s.ratio[1] / r2), math.Log(s.ratio[2] / r3), s
	}
	var sol *beamSolution
	f1, f2, sol := eval(depth, width)
	norm := math.Hypot(f1, f2)
	for range 40 {
		if norm < 2e-5 {
			break
		}
		const h = 1e-4
		a1, a2, _ := eval(depth+h, width)
		b1, b2, _ := eval(depth, width+h)
		j11, j21 := (a1-f1)/h, (a2-f2)/h
		j12, j22 := (b1-f1)/h, (b2-f2)/h
		det := j11*j22 - j12*j21
		if math.Abs(det) < 1e-12 {
			break
		}
		dd := (f1*j22 - f2*j12) / det
		dw := (j11*f2 - j21*f1) / det
		// backtrack until the residual falls
		moved := false
		for step := 1.0; step > 1e-3; step /= 2 {
			nd := max(0.02, min(depth-step*dd, 0.97))
			nw := max(0.08, min(width-step*dw, 0.9))
			g1, g2, s := eval(nd, nw)
			if n := math.Hypot(g1, g2); n < norm {
				depth, width, f1, f2, norm, sol, moved = nd, nw, g1, g2, n, s, true
				break
			}
		}
		if !moved {
			break
		}
	}
	return depth, width, sol
}

// archBar is a bar tuned by its arch: the solutions on the undercut grid
type archBar struct {
	depth, width, e float64
	grid            [archSteps + 1]*beamSolution
}

// buildArch tunes an arch to the ratios r2, r3 and solves the grid between a uniform bar and it
func buildArch(r2, r3, e, depth, width float64) *archBar {
	d, w, _ := tuneArch(r2, r3, e, depth, width)
	a := &archBar{depth: d, width: w, e: e}
	sub := newSubspace()
	for i := range a.grid {
		a.grid[i] = solveBeam(archProfile(d*float64(i)/archSteps, w, e), sub)
	}
	return a
}

// pair is the grid solutions around the undercut u and the weight of the upper
func (a *archBar) pair(u float64) (*beamSolution, *beamSolution, float64) {
	f := max(0, min(u, 1)) * archSteps
	i := int(f)
	if i >= archSteps {
		i = archSteps - 1
	}
	return a.grid[i], a.grid[i+1], f - float64(i)
}

// ratio is the frequency of mode k over the first at undercut u
func (a *archBar) ratio(k int, u float64) float64 {
	lo, hi, t := a.pair(u)
	return lo.ratio[k] + (hi.ratio[k]-lo.ratio[k])*t
}

// shape is how much a blow at pos wakes mode k at undercut u
func (a *archBar) shape(k int, pos, u float64) float64 {
	lo, hi, t := a.pair(u)
	x := max(0.01, min(pos, 1))
	return math.Abs(lo.at(k, x)*(1-t) + hi.at(k, x)*t)
}
