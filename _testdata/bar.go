package sample

func Bar(v int) bool {
	if v < 2 {
		return false
	}

	if v == 5 {
		return true
	}

	if v > 10 {
		return true
	}

	return false
}
