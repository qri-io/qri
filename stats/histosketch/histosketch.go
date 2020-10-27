// Package histosketch introduces the histosketch implementation based on
// https://github.com/aaw/histosketch
// histogram_sketch is an implementation of the Histogram Sketch data structure
// described in Ben-Haim and Tom-Tov's "A Streaming Parallel Decision Tree
// Algorithm" in Journal of Machine Learning Research 11
// (http://www.jmlr.org/papers/volume11/ben-haim10a/ben-haim10a.pdf).
//
// Modifications from Ben-Haim and Tom-Tov's original description in this
// implementation include:
//
//   * Adaptation of the "Uniform" function described in the paper into a
//     "Quantile" function here.
//
//   * Allowing initial set of centroids to be bootstrapped with the optimal
//     1-D centroid decomposition based on squared distance to centroid mean
//     via dynamic programming (see Bellman's "A note on cluster analysis and
//     dynamic programming" in Mathematical Biosciences, 18(3-4):311 – 312,
//     1973 or Haizhou Wang and Mingzhou Song's "Ckmeans.1d.dp: Optimal k-means
//     clustering in one dimension by dynamic programming" in R Journal, 3(2),
//     2011 (http://journal.r-project.org/archive/2011-2/RJournal_2011-2_Wang+Song.pdf)
//
//   * Storing the min and max value for better estimation of extreme
//     quantiles.
//
//   * Returning exact values for Sum and Quantile when they're known (before
//     any centroid merging happens).
//
//   * Improvements in handling some boundary conditions.
package histosketch

import (
	"bytes"
	"fmt"
	"math"
	"sort"
)

// Following the Ben-Haim and Tom-Tov paper, each centroid is represented by its
// value (p) and its count (m). We represent any centroids that have been
// merged with a negative m so that we can flag unmerged (exact) centroids and
// use them to return exact results where possible.
type centroid struct {
	p float64
	m float64
}

func (c *centroid) Merge(d centroid) {
	s := math.Abs(c.m) + math.Abs(d.m)
	c.p = (c.p*math.Abs(c.m) + d.p*math.Abs(d.m)) / s
	c.m = -s
}

// Sketch encapsulates the data for a probabalistic/sketch histogram implementation
type Sketch struct {
	cs    []centroid
	count float64
	min   float64
	max   float64
}

// New creates a new Sketch with a maximum of n centroids.
func New(n uint) *Sketch {
	if n == 0 {
		panic("Number of centroids must be at least 1.")
	}
	return &Sketch{make([]centroid, 0, n+1), 0, math.MaxFloat64, -math.MaxFloat64}
}

// Count returns the total number of values added to the Histogram.
func (h Sketch) Count() float64 {
	return h.count
}

// Max returns the maximum value added to the Histogram.
func (h Sketch) Max() float64 {
	return h.max
}

// Min returns the minimum value added to the Histogram.
func (h Sketch) Min() float64 {
	return h.min
}

// Add a single value to the Histogram.
func (h *Sketch) Add(value float64) {
	h.AddMany(value, 1)
}

// AddMany is equivalent to calling Add(value) count times.
func (h *Sketch) AddMany(value float64, count int64) {
	if value < h.min {
		h.min = value
	}
	if value > h.max {
		h.max = value
	}

	// Find the index k in the sorted list of centroids where (value, count) belongs.
	k := sort.Search(len(h.cs), func(i int) bool { return h.cs[i].p >= value })
	if k < len(h.cs) && h.cs[k].p == value {
		// There's an exact match for the value. Just merge the counts and return.
		h.cs[k].m += math.Copysign(1.0, h.cs[k].m) * float64(count)
		h.count += float64(count)
		return
	}

	// Move everything starting at index k up one slot in the array, insert the
	// new centroid and update the total sum of all centroid counts.
	h.cs = append(h.cs, centroid{0, 0})
	for i := len(h.cs) - 1; i > k; i-- {
		h.cs[i] = h.cs[i-1]
	}
	h.cs[k] = centroid{value, float64(count)}
	h.count += float64(count)
	if len(h.cs) < cap(h.cs) {
		// No need to merge any centroids, we're not at capacity yet.
		return
	}

	// Find the index mi such that |h.cs[mi].p - h.cs[mi+1].p| is minimized.
	mi := 0
	md := math.MaxFloat64
	for i := 0; i < len(h.cs)-1; i++ {
		d := h.cs[i+1].p - h.cs[i].p
		if d < md {
			mi, md = i, d
		}
	}

	// Merge h.cs[mi] and h.cs[mi+1], move every centroid after h.cs[mi+1]
	// down one slot in the array to fill the freed space.
	h.cs[mi].Merge(h.cs[mi+1])
	for i := mi + 1; i < len(h.cs)-1; i++ {
		h.cs[i] = h.cs[i+1]
	}
	h.cs = h.cs[:len(h.cs)-1]
}

// Merge merges two Sketches together.
func (h *Sketch) Merge(x Sketch) {
	for _, c := range x.cs {
		h.AddMany(c.p, int64(math.Abs(c.m)))
	}
}

// Gets the (i-1)-st and ith centroids. If i is 0, fake out the (i-1)-st with
// the minimum value we've seen. If i is len(h.cs), fake out the ith with the
// maximum value we've seen.
func (h Sketch) getcentroids(i int) (x, y centroid) {
	if i == 0 {
		x, y = centroid{h.min, 0.0}, h.cs[0]
	} else if i == len(h.cs) {
		x, y = h.cs[len(h.cs)-1], centroid{h.max, 0.0}
	} else {
		x, y = h.cs[i-1], h.cs[i]
	}
	return
}

// Sum returns an estimate of the number of values <= p in the histogram
func (h Sketch) Sum(p float64) float64 {
	if p >= h.max {
		return h.count
	} else if p < h.min {
		return 0.0
	}
	k := sort.Search(len(h.cs), func(i int) bool { return h.cs[i].p > p })
	s := 0.0
	for i := 0; i < k-1; i++ {
		s += math.Abs(h.cs[i].m)
	}
	ci, cj := h.getcentroids(k)
	if ci.m > 0 && (cj.m > 0 || ci.p == p) {
		return s + ci.m
	}
	mi, mj := math.Abs(ci.m), math.Abs(cj.m)
	mb := mi + (mj-mi)*(p-ci.p)/(cj.p-ci.p)
	return s + mi/2.0 + (mi+mb)*(p-ci.p)/2.0/(cj.p-ci.p)
}

// Quantile returns an estimate of the pth quantile of the histogram for 0.0 <= p <= 1.0
// pth quantile is defined as the smallest data point d such that p*total elements are
// less than or equal to d.
func (h Sketch) Quantile(p float64) float64 {
	if p < 0.0 || p > 1.0 {
		panic("bad argument, Quantile(x) only defined for real x in [0.0, 1.0]")
	}
	if h.count < 1 {
		panic("Need at least one data point to estimate Quantile")
	}
	t := p * float64(h.count)

	// Find the last two consecutive centroids ci, cj such that the sum of
	// all of the centroid weights up to and including ci plus half of cj's
	// weight is at most t. This means that the pth quantile is somewhere
	// between ci and cj.
	s := 0.0
	i := 0
	pv := 0.0
	for ; i < len(h.cs); i++ {
		v := math.Abs(h.cs[i].m) / 2.0
		if s+v+pv > t {
			break
		}
		s += v + pv
		pv = v
	}
	ci, cj := h.getcentroids(i)

	// If the two centroids are exact, return the smaller centroid value.
	if ci.m > 0 && cj.m > 0 {
		return cj.p
	}

	// Otherwise, solve for u such that t-s = (ci.m + mu)/2 * (u-ci.p)/(cj.p - ci.p),
	// where mu = ci.m + (u-ci.p)*(cj.m - ci.m)/(cj.p - ci.p). You can solve for such
	// a u using the quadratic equation so long as ci.m != cj.m, in which case the
	// solution is a little bit simpler. See Algorithm 4 and its description in the
	// original paper.
	d := t - s
	cim := math.Abs(ci.m)
	cjm := math.Abs(cj.m)
	a := cjm - cim
	if a == 0 {
		return ci.p + (cj.p-ci.p)*(d/cim)
	}
	b := 2.0 * cim
	c := -2.0 * d
	z := (-b + math.Sqrt(b*b-4*a*c)) / (2 * a)
	return ci.p + (cj.p-ci.p)*z
}

// MarshalBinary implements serialization for use with gob/encoding. See tests for an example.
func (h Sketch) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	// Version number, hardcoded to 1 for now.
	fmt.Fprintln(&b, 1)
	// Number of centroids.
	fmt.Fprintln(&b, len(h.cs))
	// centroids (value + count for each).
	for _, c := range h.cs {
		fmt.Fprintln(&b, c.p, c.m)
	}
	// Min and max observed values.
	fmt.Fprintln(&b, h.min, h.max)
	return b.Bytes(), nil
}

// UnmarshalBinary implements deserialization for use with gob/encoding. See tests for an example.
func (h *Sketch) UnmarshalBinary(data []byte) error {
	b := bytes.NewBuffer(data)
	var num, version int
	// Read and verify the version number.
	_, err := fmt.Fscanln(b, &version)
	if err != nil {
		return err
	} else if version != 1 {
		return fmt.Errorf("Unknown serialization version '%v'", version)
	}
	// Read the number of centroids.
	_, err = fmt.Fscanln(b, &num)
	if err != nil {
		return err
	}
	h.cs = make([]centroid, 0, num+1)
	// Read all centroids.
	h.count = 0.0
	for i := 0; i < num; i++ {
		c := centroid{0.0, 0.0}
		_, err := fmt.Fscanln(b, &c.p, &c.m)
		if err != nil {
			return err
		}
		h.cs = append(h.cs, c)
		h.count += c.m
	}
	// Read min and max.
	_, err = fmt.Fscanln(b, &h.min, &h.max)
	if err != nil {
		return err
	}
	return nil
}

// NewFromSample uses dynamic programming to figure out the optimal decomposition of the given
// sample into n centroids, based on the sum of squared distances from each
// point to its closest centroid. As described in Wang and Song's paper (see
// header comment for reference), the implementation is based on the following
// recurrence:
//
//   D[i,m] = min_{m <= j <= i} { D[j-1,m-1] + d(x_j, ..., x_i) },
//            1 <= i <= n, 1 <= m <= k
//
// Where D[i,m] is the optimal decomposition of the first i values in the sample
// into m centroids and d(x_1, ..., x_n) is the sum of squared distances of x_1
// through x_n to their mean.
func NewFromSample(sample []float64, n int) *Sketch {
	sort.Float64s(sample)
	if n < 1 {
		panic("Number of centroids must be at least 1")
	} else if n > len(sample) {
		// Why are you using the NewFromSample method if want more
		// centroids than the samples you have? Oh well, we'll do the
		// right thing anyway.
		centroids := make([]centroid, 0, n+1)
		for i, x := range sample {
			if i > 0 && sample[i] == sample[i-1] {
				centroids[len(centroids)-1].m++
			} else {
				centroids = append(centroids, centroid{x, 1})
			}
		}
		for _, c := range centroids {
			if c.m > 1 {
				c.m = -c.m
			}
		}
		return &Sketch{centroids, float64(len(sample)), sample[0], sample[len(sample)-1]}
	}
	// Initialize tables used for dynamic programming.
	// d[i][j] == Minimum sum of squared distances to centroid for a
	//            decomposition of the first i+1 items into j+1 centroids.
	// b[i][j] == First point in the jth centroid. Used to backtrack and
	//            create the centroid decomposition after the d matrix is
	//            filled, starting at b[len(sample)-1][n-1].
	b := make([][]int, len(sample))
	d := make([][]float64, len(sample))
	for i := range d {
		b[i] = make([]int, n)
		d[i] = make([]float64, n)
	}
	// Initialize d[i][0], the minimum sum of squared distances of the
	// first i+1 items into a single centroid using Welford's method.
	id, iu := 0.0, 0.0
	for i := 0; i < len(d); i++ {
		id += float64(i) * (sample[i] - iu) * (sample[i] - iu) / float64(i+1)
		iu = (sample[i] + float64(i)*iu) / float64(i+1)
		d[i][0] = id
	}

	for m := 1; m < n; m++ {
		for i := m; i < len(sample) && i >= m; i++ {
			// Compute sums of squared distances iteratively using
			// Welford's method.
			dist := make([]float64, i-m+1)
			id, iu := 0.0, 0.0
			for j := i; j >= m; j-- {
				idx := i - j
				diff := sample[j] - iu
				id += float64(idx) * (diff * diff) / float64(idx+1)
				iu = (sample[j] + float64(idx)*iu) / float64(idx+1)
				dist[j-m] = id
			}
			// Compute d[m][i] and b[m][i]
			mv, mj := math.MaxFloat64, i
			for j := m; j <= i; j++ {
				val := d[j-1][m-1] + dist[j-m]
				if val < mv {
					mj, mv = j, val
				}
			}
			d[i][m] = mv
			b[i][m] = mj
		}
	}

	// Create the centroid decomposition by backtracking through the b matrix.
	h := &Sketch{make([]centroid, n, n+1), float64(len(sample)), sample[0], sample[len(sample)-1]}
	centroid := n - 1
	i := len(sample) - 1
	for centroid >= 0 {
		start := b[i][centroid]
		sum := 0.0
		count := float64(i - start + 1)
		for ; i >= start; i-- {
			sum += sample[i]
		}
		p := sum / count
		h.cs[centroid].p = p
		// Need to negate counts of centroids with counts > 1
		// for compatibility with the behavior of the rest of
		// the code which uses negative counts to signal
		// merged centroids. We could be a little more careful
		// here and only negate if the centroid represents
		// an average of non-equal values.
		if count == 1 {
			h.cs[centroid].m = 1
		} else {
			h.cs[centroid].m = -count
		}
		centroid--
	}
	return h
}

// Median returns the approximate median value of the sketch
func (h *Sketch) Median() float64 {
	return h.Quantile(0.4999999)
}

func (h *Sketch) Read() ([]float64, []float64) {
	bins := make([]float64, len(h.cs)+1)
	histogram := make([]float64, len(h.cs))
	for i := 0; i < len(h.cs); i++ {
		bins[i] = h.cs[i].p
		histogram[i] = h.cs[i].m
	}
	bins[len(h.cs)] = h.max + 1
	return bins, histogram
}
