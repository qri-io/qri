package histosketch

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
	"math/rand"
	"testing"
)

func TestSortedAdd(t *testing.T) {
	s := New(32)
	s.Add(100.0)
	s.Add(101.0)
	s.Add(90.0)
	s.AddMany(92, 27)
	s.Add(99.0)
	s.Add(101.0)
	s.Add(102.0)
	s.Add(1.0)
	s.AddMany(2.0, 3)
	s.AddMany(2.0, 5)
	s.Add(2.0)
	actual := fmt.Sprintf("%v", *s)
	expected := "{[{1 1} {2 9} {90 1} {92 27} {99 1} {100 1} {101 2} {102 1}] 43 1 102}"
	if actual != expected {
		t.Errorf("Want %v, got %v", expected, actual)
	}
}

func TestMinAndMax(t *testing.T) {
	s := New(4)
	values := []float64{100.0, 50.0, 75.0, 10.0, 150.0, 5.0, 1.0, 3.0, 4.0, 300.0, 0.01, 0.1}
	for i, value := range values {
		s.Add(value)
		hmin := s.Min()
		hmax := s.Max()
		min := values[0]
		max := values[0]
		for _, x := range values[0 : i+1] {
			if x < min {
				min = x
			}
			if x > max {
				max = x
			}
		}
		if hmin != min {
			t.Errorf("Want %v for Min, got %v", min, hmin)
		}
		if hmax != max {
			t.Errorf("Want %v for Max, got %v", max, hmax)
		}
	}
}

func TestSumExact(t *testing.T) {
	s := New(16)
	for _, value := range []float64{1.0, 2.0, 3.0, 4.0, 4.0, 4.0, 5.0, 6.0} {
		s.Add(value)
	}
	if s.Sum(0.0) != 0.0 {
		t.Errorf("Want 0.0 for s.Sum(0.0), got %v", s.Sum(0.0))
	}
	if s.Sum(1.0) != 1.0 {
		t.Errorf("Want 1.0 for s.Sum(1.0), got %v", s.Sum(1.0))
	}
	if s.Sum(1.5) != 1.0 {
		t.Errorf("Want 1.0 for s.Sum(1.5), got %v", s.Sum(1.5))
	}
	if s.Sum(2.9) != 2.0 {
		t.Errorf("Want 2.0 for s.Sum(2.9), got %v", s.Sum(2.9))
	}
	if s.Sum(4.0) != 6.0 {
		t.Errorf("Want 6.0 for s.Sum(4.0), got %v", s.Sum(4.0))
	}
	if s.Sum(4.5) != 6.0 {
		t.Errorf("Want 6.0 for s.Sum(4.5), got %v", s.Sum(4.5))
	}
	if s.Sum(7.0) != 8.0 {
		t.Errorf("Want 8.0 for s.Sum(7.0), got %v", s.Sum(7.0))
	}
	if s.Sum(0.9) > s.Sum(1.0) || s.Sum(1.0) > s.Sum(1.1) {
		t.Errorf("Decreasing sums for 0.9, 1.0, 1.1: %v, %v, %v", s.Sum(0.9), s.Sum(1.0), s.Sum(1.1))
	}
	if s.Sum(3.9) > s.Sum(4.0) || s.Sum(4.0) > s.Sum(4.1) {
		t.Errorf("Decreasing sums for 3.9, 4.0, 4.1: %v, %v, %v", s.Sum(3.9), s.Sum(4.0), s.Sum(4.1))
	}
	if s.Sum(5.8) > s.Sum(5.9) || s.Sum(5.9) > s.Sum(6.0) {
		t.Errorf("Decreasing sums for 5.8, 5.9, 6.0: %v, %v, %v", s.Sum(5.8), s.Sum(5.9), s.Sum(6.0))
	}
}

func TestQuantileExact(t *testing.T) {
	s := New(16)
	for _, value := range []float64{0.01, 0.1, 1.0, 2.0, 3.0, 4.0, 4.0, 4.0, 4.1, 4.2, 4.3, 4.3} {
		s.Add(value)
	}
	if s.Quantile(0.0) != 0.01 {
		t.Errorf("Want 0.01 for s.Quantile(0.0), got %v", s.Quantile(0.0))
	}
	if s.Quantile(0.25) != 2.0 {
		t.Errorf("Want 2.0 for s.Quantile(0.25), got %v", s.Quantile(0.25))
	}
	if s.Quantile(0.5) != 4.0 {
		t.Errorf("Want 4.0 for s.Quantile(0.5), got %v", s.Quantile(0.5))
	}
	if s.Quantile(0.75) != 4.2 {
		t.Errorf("Want 4.2 for s.Quantile(0.75), got %v", s.Quantile(0.75))
	}
	if s.Quantile(1.0) != 4.3 {
		t.Errorf("Want 4.3 for s.Quantile(1.0), got %v", s.Quantile(1.0))
	}
	qs := []float64{0.001, 0.01, 0.1, 0.2, 0.4, 0.49, 0.5, 0.51, 0.75, 0.8, 0.9, 0.99, 0.999, 0.9999}
	for i := 1; i < len(qs); i++ {
		if s.Quantile(qs[i-1]) > s.Quantile(qs[i]) {
			t.Errorf("Want monotonic non-decreasing quantiles, got s.Quantile(%v) = %v and "+
				"s.Quantile(%v) = %v", qs[i-1], s.Quantile(qs[i-1]), qs[i], s.Quantile(qs[i]))
		}
	}
}

func TestQuantileArgTooSmall(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic on call to Quantile with argument < 0.0")
		}
	}()
	s := New(1024)
	s.Add(1.0)
	s.Add(2.0)
	s.Add(3.0)
	s.Quantile(-0.1)
}

func TestQuantileArgTooLarge(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic on call to Quantile with argument > 1.0")
		}
	}()
	s := New(1024)
	s.Add(1.0)
	s.Add(2.0)
	s.Add(3.0)
	s.Quantile(1.1)
}

func TestGaussianSums(t *testing.T) {
	// The relative errors here are just some arbitrary benchmarks on fixed seeds.
	// Adjusting the error or domain tested is fine, they're just meant to be
	// indicators of how much a change affects a small sketch on a real distribution.
	for seed := 0; seed < 10; seed++ {
		s := New(16)    // 2.2 KB sketch
		x := New(10000) // This histogram will be exact.
		rand.Seed(int64(seed))
		for i := 0; i < 10000; i++ {
			r := rand.NormFloat64()
			s.Add(r)
			x.Add(r)
		}
		for _, q := range []float64{-2.0, -1.0, 0.0, 1.0, 2.0, 3.0, 3.5} {
			e := math.Abs(s.Sum(q)-x.Sum(q)) / x.Sum(q)
			if e >= 0.1 {
				t.Errorf("Got error %v for s.Sum(%v) (got %v, want %v with seed = %v)",
					e, q, s.Sum(q), x.Sum(q), seed)
			}
		}
		// Sum(-3.5) on these values should be something in the single digits but the errors
		// can be a little bigger here.
		e := math.Abs(s.Sum(-3.5) - x.Sum(-3.5))
		if e > 2.0 {
			t.Errorf("Got error %v for s.Sum(-3.5) (got %v, want %v with seed = %v)",
				e, s.Sum(-3.5), x.Sum(-3.5), seed)
		}
	}
}

func TestGaussianQuantiles(t *testing.T) {
	// The relative errors here are just some arbitrary benchmarks on fixed seeds.
	// Adjusting the error or domain tested is fine, they're just meant to be
	// indicators of how much a change affects a small sketch on a real distribution.
	for seed := 0; seed < 10; seed++ {
		s := New(16)    // 2.2 KB sketch
		x := New(10000) // This histogram will be exact.
		rand.Seed(int64(seed))
		for i := 0; i < 10000; i++ {
			r := rand.NormFloat64()
			s.Add(r)
			x.Add(r)
		}
		for _, q := range []float64{0.0001, 0.001, 0.01, 0.1, 0.25, 0.35, 0.65, 0.75, 0.9, 0.99, 0.999, 0.9999} {
			e := math.Abs(s.Quantile(q)-x.Quantile(q)) / x.Quantile(q)
			if e >= 0.09 {
				t.Errorf("Got error %v for s.Quantile(%v) (got %v, want %v with seed = %v)",
					e, q, s.Quantile(q), x.Quantile(q), seed)
			}
		}
	}
}

func TestMarshal(t *testing.T) {
	s := New(16)
	for _, value := range []float64{1.0, 2.0, 3.0, 4.0, 4.0, 4.0, 5.0, 6.0} {
		s.Add(value)
	}
	want := fmt.Sprintf("%v", s)
	var stream bytes.Buffer
	enc := gob.NewEncoder(&stream)
	err := enc.Encode(s)
	if err != nil {
		t.Errorf("Got error encoding Sketch: %v", err)
	}
	dec := gob.NewDecoder(&stream)
	s2 := &Sketch{}
	err = dec.Decode(&s2)
	if err != nil {
		t.Errorf("Got error decoding serialized Sketch: %v", err)
	}
	got := fmt.Sprintf("%v", s2)
	if want != got {
		t.Errorf("After unmarshalling, got %v, want %v", got, want)
	}
}

func TestOptimalOneDataPoint(t *testing.T) {
	got := NewFromSample([]float64{1.0}, 1)
	want := "&{[{1 1}] 1 1 1}"
	if fmt.Sprintf("%v", got) != want {
		t.Errorf("Want %v, got %v", want, got)
	}
}

func TestOptimalOneCentroid(t *testing.T) {
	got := NewFromSample([]float64{1.0, 2.0, 3.0, 4.0, 5.0}, 1)
	want := "&{[{3 -5}] 5 1 5}"
	if fmt.Sprintf("%v", got) != want {
		t.Errorf("Want %v, got %v", want, got)
	}
}

func TestOptimalTwoCentroids(t *testing.T) {
	got := NewFromSample([]float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}, 2)
	want := "&{[{2 -3} {5 -3}] 6 1 6}"
	if fmt.Sprintf("%v", got) != want {
		t.Errorf("Want %v, got %v", want, got)
	}
}

func TestOptimalTwoCentroidsSkewed(t *testing.T) {
	got := NewFromSample([]float64{1.0, 7.0, 7.0, 8.0, 8.0, 9.0, 9.0}, 2)
	want := "&{[{1 1} {8 -6}] 7 1 9}"
	if fmt.Sprintf("%v", got) != want {
		t.Errorf("Want %v, got %v", want, got)
	}
}

func TestOptimalManyCentroids(t *testing.T) {
	got := NewFromSample([]float64{1.0, 7.0, 7.0, 8.0, 8.0, 9.0, 9.0}, 7)
	want := "&{[{1 1} {7 1} {7 1} {8 1} {8 1} {9 1} {9 1}] 7 1 9}"
	if fmt.Sprintf("%v", got) != want {
		t.Errorf("Want %v, got %v", want, got)
	}
}

func TestOptimalMaxCentroids(t *testing.T) {
	got := NewFromSample([]float64{1.0, 2.0, 3.0, 4.0, 4.0, 5.0, 6.0}, 6)
	want := "&{[{1 1} {2 1} {3 1} {4 -2} {5 1} {6 1}] 7 1 6}"
	if fmt.Sprintf("%v", got) != want {
		t.Errorf("Want %v, got %v", want, got)
	}
}

func TestOptimalTooFewCentroids(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic on call to NewFromSample with 0 centroids")
		}
	}()
	NewFromSample([]float64{1.0, 2.0, 3.0, 4.0, 4.0, 5.0, 6.0}, 0)
}

func TestOptimalTooManyCentroids(t *testing.T) {
	got := NewFromSample([]float64{1.0, 2.0, 3.0, 4.0, 4.0, 5.0, 6.0}, 10)
	want := "&{[{1 1} {2 1} {3 1} {4 2} {5 1} {6 1}] 7 1 6}"
	if fmt.Sprintf("%v", got) != want {
		t.Errorf("Want %v, got %v", want, got)
	}
}

func TestMedian(t *testing.T) {
	sketch := NewFromSample([]float64{1.0, 2.0, 3.0, 4.0, 4.0, 5.0, 6.0}, 10)
	got := sketch.Median()
	want := 4.0
	if got != want {
		t.Errorf("Want %v, got %v", want, got)
	}

	sketch = NewFromSample([]float64{1.0, 2.0, 2.0, 3.0, 4.0, 5.0, 6.0}, 10)
	got = sketch.Median()
	want = 3.0
	if got != want {
		t.Errorf("Want %v, got %v", want, got)
	}

	sketch = NewFromSample([]float64{2.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0}, 10)
	got = sketch.Median()
	want = 3.0 // median is probabalistic and skews to the right
	if got != want {
		t.Errorf("Want %v, got %v", want, got)
	}
}

func TestRead(t *testing.T) {
	sketch := NewFromSample([]float64{1.0, 2.0, 3.0, 4.0, 4.0, 5.0, 6.0}, 10)
	bins, histogram := sketch.Read()
	wantBins := "[1 2 3 4 5 6 7]"
	wantHistogram := "[1 1 1 2 1 1]"
	if fmt.Sprintf("%v", bins) != wantBins {
		t.Errorf("Want bins %v, got %v", wantBins, bins)
	}
	if fmt.Sprintf("%v", histogram) != wantHistogram {
		t.Errorf("Want histogram %v, got %v", wantHistogram, histogram)
	}
}
