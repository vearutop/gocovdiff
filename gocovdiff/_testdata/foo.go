package sample

var i = 15

func foo(v int) bool {
	if v == i {
		return false
	}

	if v < 2 {
		return false
	}

	if v > 10 {
		return true
	}

	if v == 6 {
		return true
	}

	return false
}
