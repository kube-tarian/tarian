package tarianpb

func NewConstraint() *Constraint {
	c := &Constraint{Kind: KindConstraint}
	return c
}

func NewAction() *Action {
	a := &Action{Kind: KindAction}
	return a
}

func NewEvent() *Event {
	e := &Event{Kind: KindEvent}
	return e
}
