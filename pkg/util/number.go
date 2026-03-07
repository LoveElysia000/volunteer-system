package util

// ClampInt32 clamps a positive int into int32 range.
func ClampInt32(v int) int32 {
	if v <= 0 {
		return 0
	}
	if v > int(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(v)
}
