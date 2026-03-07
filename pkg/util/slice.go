package util

// UniquePositiveInt64 returns unique positive ids while preserving first-seen order.
func UniquePositiveInt64(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}

	uniqueIDs := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	return uniqueIDs
}

// ContainsInt64 reports whether target exists in items.
func ContainsInt64(items []int64, target int64) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// ReverseInPlace reverses the slice in-place.
func ReverseInPlace[T any](items []T) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}
