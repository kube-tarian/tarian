package tarianpb

func NewConstraint() *Constraint {
	c := &Constraint{Kind: KindConstraint}
	return c
}

func NewEvent() *Event {
	e := &Event{Kind: KindEvent}
	return e
}
