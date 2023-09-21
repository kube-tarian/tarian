package tarianpb

// ToString converts a FalcoPriority enum value to its string representation.
func (f FalcoPriority) ToString() string {
	return FalcoPriority_name[int32(f)]
}

// FalcoPriorityFromString converts a string representation of a FalcoPriority
func FalcoPriorityFromString(s string) FalcoPriority {
	return FalcoPriority(FalcoPriority_value[s])
}
