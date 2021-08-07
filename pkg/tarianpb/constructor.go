package tarianpb

func NewConstraint() *Constraint {
	c := &Constraint{Kind: KindConstraint}
	return c
}
