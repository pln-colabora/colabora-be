package rbac

import (
	"errors"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
)

var ErrWorkflowForbidden = errors.New("caller does not own workflow node")

// OwnsWorkflowNode resolves role and unit ownership independently of node state.
// Use AuthorizeWorkflowNode for write authorization, including prerequisite gates.
func OwnsWorkflowNode(role, unit, connection, ulp string, code workflow.Code) bool {
	owner, ok := workflow.Owner(code, connection)
	if !ok || role == RoleSuperUser || role != owner {
		return false
	}
	if role == RoleTeknik || role == RolePelayananPelanggan {
		return unit != "" && ulp != "" && unit == ulp
	}
	return true
}

func AuthorizeWorkflowNode(role, unit, ulp string, s workflow.Snapshot, code workflow.Code) error {
	if !OwnsWorkflowNode(role, unit, s.Connection, ulp, code) {
		return ErrWorkflowForbidden
	}
	r, err := workflow.Evaluate(s)
	if err != nil {
		return err
	}
	if r.Status != workflow.Active || (r.Nodes[code] != workflow.Available && r.Nodes[code] != workflow.InProgress) {
		return workflow.ErrNotActionable
	}
	return nil
}

// AvailableActions returns caller-owned node metadata in canonical order.
// HTTP adapters supply method/path and combine nodes for bundled endpoints.
func AvailableActions(role, unit, ulp string, s workflow.Snapshot) ([]workflow.Definition, error) {
	r, err := workflow.Evaluate(s)
	if err != nil {
		return nil, err
	}
	out := []workflow.Definition{}
	for _, code := range r.AvailableNodes {
		if OwnsWorkflowNode(role, unit, s.Connection, ulp, code) {
			d, _ := workflow.Lookup(code)
			out = append(out, d)
		}
	}
	return out, nil
}
