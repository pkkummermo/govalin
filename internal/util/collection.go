package util

func ContainsSome[T comparable](collection []T, candidates ...T) bool {
	for _, item := range collection {
		for _, candidate := range candidates {
			if item == candidate {
				return true
			}
		}
	}
	return false
}

func All[T comparable](slice []T, pred func(T) bool) bool {
	for _, t := range slice {
		if !pred(t) {
			return false
		}
	}
	return true
}
