package workflow

import (
	"errors"
	"fmt"
)

type Status string

const (
	Locked     Status = "locked"
	Available  Status = "available"
	InProgress Status = "in_progress"
	Completed  Status = "completed"
	Skipped    Status = "skipped"
)

type AggregateStatus string

const (
	Active   AggregateStatus = "in_progress"
	Finished AggregateStatus = "completed"
	Returned AggregateStatus = "returned"
)

type Delegation string

const (
	Delegated Delegation = "delegated"
	Return    Delegation = "returned"
)

var (
	ErrInvalidState  = errors.New("invalid workflow state")
	ErrNotActionable = errors.New("workflow node is not actionable")
)

type Decisions struct {
	KebutuhanTiang *bool
	PerluPDKB      *bool
	NPS            Delegation
}

// Snapshot contains persisted facts. Missing nodes are treated as locked;
// available/locked/skipped projections are recomputed, never trusted as gates.
// Decision values may be staged before completion but take effect only after
// their owning node completes. Completion requires its decision to be present.
type Snapshot struct {
	Connection string
	Nodes      map[Code]Status
	Decisions  Decisions
}
type Result struct {
	Nodes          map[Code]Status
	Status         AggregateStatus
	CurrentStage   int16  // 7 when completed, 3 when returned.
	AvailableNodes []Code // Includes resumable in_progress nodes.
}

func Evaluate(s Snapshot) (Result, error) {
	fail := func(detail string) (Result, error) { return Result{}, fmt.Errorf("%w: %s", ErrInvalidState, detail) }
	if !ValidConnection(s.Connection) {
		return fail("unknown connection type")
	}
	if s.Decisions.NPS != "" && s.Decisions.NPS != Delegated && s.Decisions.NPS != Return {
		return fail("unknown NPS outcome")
	}
	for code, status := range s.Nodes {
		if _, ok := Lookup(code); !ok {
			return fail("unknown node " + string(code))
		}
		switch status {
		case Locked, Available, InProgress, Completed, Skipped:
		default:
			return fail("unknown node status")
		}
	}
	for _, check := range []struct {
		code    Code
		missing bool
	}{
		{KebutuhanTiang, s.Decisions.KebutuhanTiang == nil},
		{WOKonstruksi, s.Decisions.PerluPDKB == nil},
		{NPS, s.Decisions.NPS == ""},
	} {
		if s.Nodes[check.code] == Completed && check.missing {
			return fail("missing decision for " + string(check.code))
		}
	}

	r := Result{Nodes: make(map[Code]Status, len(definitions)), Status: Active, CurrentStage: 7, AvailableNodes: []Code{}}
	terminal := s.Nodes[NPS] == Completed && s.Decisions.NPS == Return
	for _, d := range definitions {
		known, applicable := true, true
		switch d.Applicability {
		case PolesRequired:
			known = r.Nodes[KebutuhanTiang] == Completed && s.Decisions.KebutuhanTiang != nil
			applicable = known && *s.Decisions.KebutuhanTiang
		case PDKBRequired:
			known = r.Nodes[WOKonstruksi] == Completed && s.Decisions.PerluPDKB != nil
			applicable = known && *s.Decisions.PerluPDKB
		}
		ready := known && applicable
		for _, dep := range d.Prerequisites {
			if r.Nodes[dep] != Completed && r.Nodes[dep] != Skipped {
				ready = false
			}
		}
		if terminal && d.Stage > 3 {
			ready = false
		}
		status := Locked
		if known && !applicable {
			status = Skipped
		} else if ready {
			status = Available
		}
		persisted := s.Nodes[d.Code]
		if persisted == Completed || persisted == InProgress {
			if !ready {
				return fail("unmet prerequisites or inapplicable node " + string(d.Code))
			}
			status = persisted
		}
		if persisted == Skipped && status != Skipped {
			return fail("invalid skip " + string(d.Code))
		}
		r.Nodes[d.Code] = status
		if status != Completed && status != Skipped && d.Stage < r.CurrentStage {
			r.CurrentStage = d.Stage
		}
		if status == Available || status == InProgress {
			r.AvailableNodes = append(r.AvailableNodes, d.Code)
		}
	}
	if terminal {
		r.Status = Returned
		r.CurrentStage = 3
		r.AvailableNodes = []Code{}
	}
	if r.Nodes[Selesai] == Completed {
		r.Status = Finished
		r.CurrentStage = 7
		r.AvailableNodes = []Code{}
	}
	return r, nil
}

// Transition starts or completes one node on a detached snapshot. Callers own
// authorization, evidence validation, transactions and audit persistence. Bundled
// endpoints can compose ordered transitions and persist only the final result.
func Transition(s Snapshot, code Code, target Status) (Snapshot, Result, error) {
	r, err := Evaluate(s)
	if err != nil {
		return Snapshot{}, Result{}, err
	}
	if target != InProgress && target != Completed {
		return Snapshot{}, Result{}, ErrInvalidState
	}
	current := r.Nodes[code]
	if r.Status != Active || (current != Available && current != InProgress) || (target == InProgress && current == InProgress) {
		return Snapshot{}, Result{}, ErrNotActionable
	}
	next := Snapshot{Connection: s.Connection, Nodes: r.Nodes, Decisions: s.Decisions}
	if next.Decisions.KebutuhanTiang != nil {
		v := *next.Decisions.KebutuhanTiang
		next.Decisions.KebutuhanTiang = &v
	}
	if next.Decisions.PerluPDKB != nil {
		v := *next.Decisions.PerluPDKB
		next.Decisions.PerluPDKB = &v
	}
	next.Nodes[code] = target
	evaluated, err := Evaluate(next)
	if err != nil {
		return Snapshot{}, Result{}, err
	}
	return next, evaluated, nil
}
