package rbac

import (
	"errors"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
)

var ErrWorkflowForbidden = errors.New("caller does not own workflow node")

// VendorWO returns the issued work-order node visible to a vendor account.
func VendorWO(role string) (workflow.Code, bool) {
	switch role {
	case RoleVendorTiang:
		return workflow.WOTiang, true
	case RoleVendorKonstruksi:
		return workflow.WOKonstruksi, true
	case RoleVendorSrApp:
		return workflow.WOAPP, true
	default:
		return "", false
	}
}

// VendorVisibleNodes returns the read projection for a vendor request. The
// issued WO is included even though its owner is an internal PLN role; the
// remaining nodes are derived from the canonical connection-specific owner
// matrix. This is a read projection only and does not grant write access.
func VendorVisibleNodes(role, connection string) []workflow.Code {
	if !IsVendor(role) || !workflow.ValidConnection(connection) {
		return []workflow.Code{}
	}

	visible := make([]workflow.Code, 0, 4)
	if issuedWO, ok := VendorWO(role); ok {
		// vendor-sr-app is only assigned for JTR/JTM. PLG TM uses the
		// vendor-konstruksi branch instead, so do not expose an unrelated WO.
		if role != RoleVendorSrApp {
			visible = append(visible, issuedWO)
		} else if owner, ok := workflow.Owner(workflow.SRAPP, connection); ok && owner == role {
			visible = append(visible, issuedWO)
		}
	}
	for _, definition := range workflow.Definitions() {
		if len(visible) > 0 && definition.Code == visible[0] {
			continue
		}
		owner, ok := workflow.Owner(definition.Code, connection)
		if ok && owner == role {
			visible = append(visible, definition.Code)
		}
	}
	return visible
}

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
