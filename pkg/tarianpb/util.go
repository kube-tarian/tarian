package tarianpb

func (f FalcoPriority) ToString() string {
	return FalcoPriority_name[int32(f)]
}

func FalcoPriorityFromString(s string) FalcoPriority {
	return FalcoPriority(FalcoPriority_value[s])
}
