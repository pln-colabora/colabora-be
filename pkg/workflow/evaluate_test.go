package workflow_test

import (
	"fmt"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
	"testing"
)

func boolptr(v bool) *bool { return &v }
func base(poles, pdkb bool) workflow.Snapshot {
	return workflow.Snapshot{Connection: "JTR", Decisions: workflow.Decisions{KebutuhanTiang: boolptr(poles), PerluPDKB: boolptr(pdkb), NPS: workflow.Delegated}}
}
func complete(t *testing.T, s workflow.Snapshot, codes ...workflow.Code) workflow.Snapshot {
	t.Helper()
	for _, code := range codes {
		next, _, err := workflow.Transition(s, code, workflow.Completed)
		require.NoError(t, err, "complete %s", code)
		s = next
	}
	return s
}
func delegated(t *testing.T, s workflow.Snapshot) workflow.Snapshot {
	return complete(t, s, workflow.Permohonan, workflow.Survei, workflow.RAB, workflow.KebutuhanTiang, workflow.Perluasan, workflow.NPS)
}

func TestLifecycle(t *testing.T) {
	for _, connection := range []string{"JTR", "JTM/Gardu", "PLG TM <5 GWNG", "PLG TM >5 GWNG"} {
		for _, poles := range []bool{false, true} {
			for _, pdkb := range []bool{false, true} {
				for _, reverse := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/poles=%v/pdkb=%v/reverse=%v", connection, poles, pdkb, reverse), func(t *testing.T) {
						s := base(poles, pdkb)
						s.Connection = connection
						s = delegated(t, s)
						// Finish independent branches in opposite orders, including nodes in later
						// stages while another branch still has stage 4 work outstanding.
						pole := func() {
							if poles {
								s = complete(t, s, workflow.WOTiang, workflow.PemasanganTiang)
							}
						}
						app := func() { s = complete(t, s, workflow.WOAPP, workflow.Reservasi, workflow.Tera) }
						construction := func() {
							s = complete(t, s, workflow.WOKonstruksi)
							if pdkb {
								s = complete(t, s, workflow.WOPDKB)
							}
							s = complete(t, s, workflow.Konstruksi)
							if pdkb {
								s = complete(t, s, workflow.DokumentasiPDKB)
							}
						}
						if reverse {
							construction()
							app()
							pole()
						} else {
							pole()
							app()
							construction()
						}
						if reverse {
							s = complete(t, s, workflow.SRAPP, workflow.Energize)
						} else {
							s = complete(t, s, workflow.Energize, workflow.SRAPP)
						}
						s = complete(t, s, workflow.PDL, workflow.AIL, workflow.Selesai)
						r, err := workflow.Evaluate(s)
						require.NoError(t, err)
						require.Equal(t, workflow.Finished, r.Status)
						require.EqualValues(t, 7, r.CurrentStage)
						require.Empty(t, r.AvailableNodes)
						for _, d := range workflow.Definitions() {
							expected := workflow.Completed
							if (!poles && (d.Code == workflow.WOTiang || d.Code == workflow.PemasanganTiang)) || (!pdkb && (d.Code == workflow.WOPDKB || d.Code == workflow.DokumentasiPDKB)) {
								expected = workflow.Skipped
							}
							require.Equal(t, expected, r.Nodes[d.Code], "%s", d.Code)
							_, _, err = workflow.Transition(s, d.Code, workflow.Completed)
							require.ErrorIs(t, err, workflow.ErrNotActionable)
						}
					})
				}
			}
		}
	}
}

func TestEveryPrerequisiteIsRequired(t *testing.T) {
	// A valid completed prefix is shortened by each direct prerequisite in turn.
	// Evaluating forged completion must fail rather than accepting impossible history.
	s := base(true, true)
	for _, d := range workflow.Definitions() {
		for _, dep := range d.Prerequisites {
			t.Run(string(d.Code)+"/"+string(dep), func(t *testing.T) {
				bad := s
				bad.Nodes = make(map[workflow.Code]workflow.Status)
				for k, v := range s.Nodes {
					bad.Nodes[k] = v
				}
				bad.Nodes[dep] = workflow.Locked
				bad.Nodes[d.Code] = workflow.Completed
				_, err := workflow.Evaluate(bad)
				require.ErrorIs(t, err, workflow.ErrInvalidState)
			})
		}
		s = complete(t, s, d.Code)
	}
}

func TestJoinGatesAndStageProjection(t *testing.T) {
	s := delegated(t, base(true, true))
	s = complete(t, s, workflow.WOKonstruksi)
	_, _, err := workflow.Transition(s, workflow.PKVendor, workflow.Completed)
	require.ErrorIs(t, err, workflow.ErrNotActionable)
	s = complete(t, s, workflow.WOPDKB, workflow.Konstruksi)
	r, err := workflow.Evaluate(s)
	require.NoError(t, err)
	require.EqualValues(t, 4, r.CurrentStage)
	require.Contains(t, r.AvailableNodes, workflow.DokumentasiPDKB)
	require.Equal(t, workflow.Locked, r.Nodes[workflow.Energize])
	require.Equal(t, workflow.Locked, r.Nodes[workflow.SRAPP])
	s = complete(t, s, workflow.WOAPP, workflow.Reservasi, workflow.Tera, workflow.SRAPP)
	r, err = workflow.Evaluate(s)
	require.NoError(t, err)
	require.Equal(t, workflow.Locked, r.Nodes[workflow.PDL])
	s = complete(t, s, workflow.DokumentasiPDKB)
	r, err = workflow.Evaluate(s)
	require.NoError(t, err)
	require.Equal(t, workflow.Locked, r.Nodes[workflow.Energize])
	s = complete(t, s, workflow.WOTiang, workflow.PemasanganTiang, workflow.Energize)
	r, err = workflow.Evaluate(s)
	require.NoError(t, err)
	require.Contains(t, r.AvailableNodes, workflow.PDL)
}

func TestReturnAndUnknownDecisions(t *testing.T) {
	for _, connection := range []string{"JTR", "JTM/Gardu", "PLG TM <5 GWNG", "PLG TM >5 GWNG"} {
		s := base(false, false)
		s.Connection = connection
		s.Decisions.NPS = workflow.Return
		s = delegated(t, s)
		r, err := workflow.Evaluate(s)
		require.NoError(t, err)
		require.Equal(t, workflow.Returned, r.Status)
		require.EqualValues(t, 3, r.CurrentStage)
		require.Empty(t, r.AvailableNodes)
		for _, code := range []workflow.Code{workflow.NPS, workflow.WOTiang, workflow.WOKonstruksi, workflow.WOAPP} {
			_, _, err = workflow.Transition(s, code, workflow.Completed)
			require.ErrorIs(t, err, workflow.ErrNotActionable)
		}
	}
	s := workflow.Snapshot{Connection: "JTR"}
	r, err := workflow.Evaluate(s)
	require.NoError(t, err)
	require.Len(t, r.Nodes, 20)
	require.Equal(t, []workflow.Code{workflow.Permohonan}, r.AvailableNodes)
	require.Equal(t, workflow.Locked, r.Nodes[workflow.WOTiang])
	require.Equal(t, workflow.Locked, r.Nodes[workflow.WOPDKB])
	s = complete(t, s, workflow.Permohonan, workflow.Survei, workflow.RAB)
	_, _, err = workflow.Transition(s, workflow.KebutuhanTiang, workflow.Completed)
	require.ErrorIs(t, err, workflow.ErrInvalidState)
}

func TestValidationAndIsolation(t *testing.T) {
	for _, s := range []workflow.Snapshot{
		{Connection: "unknown"},
		{Connection: "JTR", Nodes: map[workflow.Code]workflow.Status{"unknown": workflow.Available}},
		{Connection: "JTR", Nodes: map[workflow.Code]workflow.Status{workflow.Permohonan: "done"}},
		{Connection: "JTR", Nodes: map[workflow.Code]workflow.Status{workflow.Permohonan: workflow.Skipped}},
		{Connection: "JTR", Nodes: map[workflow.Code]workflow.Status{workflow.Survei: workflow.InProgress}},
		{Connection: "JTR", Decisions: workflow.Decisions{NPS: "approved"}},
	} {
		_, err := workflow.Evaluate(s)
		require.ErrorIs(t, err, workflow.ErrInvalidState)
	}
	s := base(false, false)
	next, r, err := workflow.Transition(s, workflow.Permohonan, workflow.InProgress)
	require.NoError(t, err)
	require.Nil(t, s.Nodes)
	require.Contains(t, r.AvailableNodes, workflow.Permohonan)
	_, _, err = workflow.Transition(next, workflow.Permohonan, workflow.InProgress)
	require.ErrorIs(t, err, workflow.ErrNotActionable)
	_, _, err = workflow.Transition(next, workflow.Permohonan, workflow.Skipped)
	require.ErrorIs(t, err, workflow.ErrInvalidState)
	*next.Decisions.KebutuhanTiang = true
	require.False(t, *s.Decisions.KebutuhanTiang)
	next = complete(t, next, workflow.Permohonan)
	r.Nodes[workflow.Permohonan] = workflow.Locked
	require.Equal(t, workflow.Completed, next.Nodes[workflow.Permohonan])
	defs := workflow.Definitions()
	*defs[0].ActivityNumber = 99
	defs[1].Prerequisites[0] = workflow.Selesai
	d, _ := workflow.Lookup(workflow.Permohonan)
	require.EqualValues(t, 1, *d.ActivityNumber)
	d, _ = workflow.Lookup(workflow.Survei)
	require.Equal(t, []workflow.Code{workflow.Permohonan}, d.Prerequisites)
	for _, code := range []workflow.Code{workflow.KebutuhanTiang, workflow.WOPDKB, workflow.DokumentasiPDKB} {
		d, _ = workflow.Lookup(code)
		require.Nil(t, d.ActivityNumber)
		require.Nil(t, d.SLAActivityNumber)
	}
	d, _ = workflow.Lookup(workflow.NPS)
	require.EqualValues(t, 5, *d.ActivityNumber)
	require.Nil(t, d.SLAActivityNumber)
}

func TestDecisionsAndPersistedProjections(t *testing.T) {
	s := delegated(t, base(false, false))
	s.Decisions.PerluPDKB = nil
	_, _, err := workflow.Transition(s, workflow.WOKonstruksi, workflow.Completed)
	require.ErrorIs(t, err, workflow.ErrInvalidState)
	s = base(false, false)
	s.Decisions.NPS = ""
	s = complete(t, s, workflow.Permohonan, workflow.Survei, workflow.RAB, workflow.KebutuhanTiang, workflow.Perluasan)
	_, _, err = workflow.Transition(s, workflow.NPS, workflow.Completed)
	require.ErrorIs(t, err, workflow.ErrInvalidState)

	// A stale available projection cannot bypass NPS, even with decisions staged.
	s = base(true, true)
	s.Nodes = map[workflow.Code]workflow.Status{workflow.WOTiang: workflow.Available}
	r, err := workflow.Evaluate(s)
	require.NoError(t, err)
	require.Equal(t, workflow.Locked, r.Nodes[workflow.WOTiang])
	_, _, err = workflow.Transition(s, workflow.WOTiang, workflow.Completed)
	require.ErrorIs(t, err, workflow.ErrNotActionable)
	s.Nodes[workflow.WOTiang] = workflow.Skipped
	_, err = workflow.Evaluate(s)
	require.ErrorIs(t, err, workflow.ErrInvalidState)
}
