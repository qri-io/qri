package datatypes

func EachIsType(typ Type, ts []Type) bool {
	for _, t := range ts {
		if typ != t {
			return false
		}
	}
	return true
}

func EachSame(ts []Type) (typ Type) {
	for i, t := range ts {
		if i == 0 {
			typ = t
		}
		if typ != t {
			return Unknown
		}
	}
	return
}

func EachNumeric(ts []Type) bool {
	for _, t := range ts {
		if t != Float && t != Integer {
			return false
		}
	}
	return true
}
