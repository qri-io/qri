package dataset

const b26chars = "abcdefghijklmnopqrstuvwxyz"

func base26(d int) (s string) {
	var cols []int
	if d == 0 {
		return "a"
	}

	for d != 0 {
		cols = append(cols, d%26)
		d = d / 26
	}
	for i := len(cols) - 1; i >= 0; i-- {
		if i != 0 && cols[i] > 0 {
			s += string(b26chars[cols[i]-1])
		} else {
			s += string(b26chars[cols[i]])
		}
	}
	return s
}
