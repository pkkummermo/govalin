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
