package model

func arrayIntersect(left, right []string) (in []string) {
	m := make(map[string]bool)

	for _, l := range left {
		m[l] = true
	}

	for _ , r := range right {
		if _, ok := m[r]; ok {
			in = append(in, r)
		}
	}

	return
}
func arrayDisjunct(left, right []string) (out []string) {
	m := make(map[string]bool)

	for _, l := range left {
		m[l] = true
	}

	for _ , r := range right {
		if _, ok := m[r]; !ok {
			out = append(out, r)
		}
	}

	return
}
func arrayKeys(m map[string]string) (out []string) {
	for k, _ := range m {
		out = append(out, k)
	}
	return
}


func arrayDisjunctInt(left, right []int) (out []int) {
	m := make(map[int]bool)

	for _, l := range left {
		m[l] = true
	}

	for _ , r := range right {
		if _, ok := m[r]; !ok {
			out = append(out, r)
		}
	}

	return
}
