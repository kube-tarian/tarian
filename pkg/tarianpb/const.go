package tarianpb

// KindConstraint represents the kind of object as "Constraint".
const KindConstraint = "Constraint"

// KindAction represents the kind of object as "Action".
const KindAction = "Action"

// KindEvent represents the kind of object as "Event".
const KindEvent = "Event"

// EventTypeViolation represents an event type for violations.
const EventTypeViolation = "violation"

// EventTypeFalcoAlert represents an event type for Falco alerts.
const EventTypeFalcoAlert = "falco_alert"

// EventTypePodDeleted represents an event type for deleted pods.
const EventTypePodDeleted = "pod_deleted"

// EventTypeDetection represents an event type for tarain-detection.
const EventTypeDetection = "tarian-detection/detection" //prefix which is coming from tarian-detector library and type
