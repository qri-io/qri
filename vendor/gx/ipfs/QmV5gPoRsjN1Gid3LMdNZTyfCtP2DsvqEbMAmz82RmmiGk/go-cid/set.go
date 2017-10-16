package cid

type Set struct {
	set map[string]struct{}
}

func NewSet() *Set {
	return &Set{set: make(map[string]struct{})}
}

func (s *Set) Add(c *Cid) {
	s.set[string(c.Bytes())] = struct{}{}
}

func (s *Set) Has(c *Cid) bool {
	_, ok := s.set[string(c.Bytes())]
	return ok
}

func (s *Set) Remove(c *Cid) {
	delete(s.set, string(c.Bytes()))
}

func (s *Set) Len() int {
	return len(s.set)
}

func (s *Set) Keys() []*Cid {
	out := make([]*Cid, 0, len(s.set))
	for k, _ := range s.set {
		c, _ := Cast([]byte(k))
		out = append(out, c)
	}
	return out
}

func (s *Set) Visit(c *Cid) bool {
	if !s.Has(c) {
		s.Add(c)
		return true
	}

	return false
}

func (s *Set) ForEach(f func(c *Cid) error) error {
	for cs, _ := range s.set {
		c, _ := Cast([]byte(cs))
		err := f(c)
		if err != nil {
			return err
		}
	}
	return nil
}
