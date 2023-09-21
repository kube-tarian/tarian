package tarianpb

// NewConstraint creates a new Constraint.
func NewConstraint() *Constraint {
	c := &Constraint{Kind: KindConstraint}
	return c
}

// NewAction creates a new Action.
func NewAction() *Action {
	a := &Action{Kind: KindAction}
	return a
}

// NewEvent creates a new Event.
func NewEvent() *Event {
	e := &Event{Kind: KindEvent}
	return e
}
