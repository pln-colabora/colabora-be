package rbac

// OwnsActivity resolves the given 1-17 activity number to its UI stage and delegates to
// OwnsStage. An unknown activity number never has an owner.
func OwnsActivity(role, unit string, activityNumber int16, jenisSambungan, ulpUnit, ownerFnOverride string) bool {
	stage, ok := ActivityStageMap[activityNumber]
	if !ok {
		return false
	}
	return OwnsStage(role, unit, stage, jenisSambungan, ulpUnit, ownerFnOverride)
}
