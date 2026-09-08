package entities

import (
	"fmt"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"time"
)

// WorkflowSnapshot adapts persisted nodes to the pure domain engine.
func (p Permohonan) WorkflowSnapshot() workflow.Snapshot {
	s := workflow.Snapshot{Connection: p.JenisSambungan, Nodes: map[workflow.Code]workflow.Status{},
		Decisions: workflow.Decisions{KebutuhanTiang: p.KebutuhanTiang, PerluPDKB: p.PerluPdkb}}
	if p.NpsDelegationStatus != nil {
		s.Decisions.NPS = workflow.Delegation(*p.NpsDelegationStatus)
	}
	for _, n := range p.WorkflowNodes {
		s.Nodes[workflow.Code(n.WorkflowNode)] = workflow.Status(n.Status)
	}
	return s
}

// InitializeWorkflow requires every numbered SLA rule; nodes without a rule
// reference receive a nil deadline, never an invented date.
func InitializeWorkflow(p Permohonan, rules []SLARule, now time.Time) ([]PermohonanActivity, workflow.Result, error) {
	s := workflow.Snapshot{Connection: p.JenisSambungan}
	_, result, err := workflow.Transition(s, workflow.Permohonan, workflow.Completed)
	if err != nil {
		return nil, workflow.Result{}, err
	}
	offsets := map[int16]int16{}
	for _, rule := range rules {
		if rule.JenisSambungan == p.JenisSambungan {
			offsets[rule.ActivityNumber] = rule.OffsetDays
		}
	}
	nodes := make([]PermohonanActivity, 0, len(workflow.Definitions()))
	for _, d := range workflow.Definitions() {
		n := PermohonanActivity{PermohonanID: p.ID, WorkflowNode: string(d.Code), ActivityNumber: d.ActivityNumber,
			StageNumber: d.Stage, Status: string(result.Nodes[d.Code]), Payload: "{}"}
		if d.SLAActivityNumber != nil {
			offset, ok := offsets[*d.SLAActivityNumber]
			if !ok {
				return nil, workflow.Result{}, fmt.Errorf("missing SLA rule for activity %d", *d.SLAActivityNumber)
			}
			deadline := p.RequestDate.AddDate(0, 0, int(offset))
			n.SlaDeadline = &deadline
		}
		if d.Code == workflow.Permohonan {
			actor := p.CreatedBy
			completed := now
			n.CompletedBy = &actor
			n.CompletedAt = &completed
		}
		nodes = append(nodes, n)
	}
	return nodes, result, nil
}
