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
