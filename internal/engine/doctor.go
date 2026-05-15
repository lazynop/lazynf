package engine

import (
	"context"
	"errors"
	"strings"

	"github.com/lazynop/lazynf/internal/doctor"
)

// RunDoctor executes the doctor diagnostic and emits one DoctorSectionEvent
// per Check, followed by a CompletedEvent. doctor.Run is synchronous, so all
// section events are emitted in order after the underlying call returns.
//
// Sections do NOT emit ConflictEvent. The Action field on each
// DoctorSectionEvent is mapped from the Check's section name when a
// well-known remedial action exists (catalog → RefreshCatalog, fc-cache →
// RefreshFontCache); other sections get ActionNone.
func (e *Engine) RunDoctor(ctx context.Context) OpHandle {
	opID := e.nextOpID()
	em := newEmitter(ctx)

	go func() {
		defer em.Close()
		em.Send(StartedEvent{OpID: opID, Kind: KindDoctor})

		params := doctor.Params{
			FontDir:     e.deps.FontDir,
			StatePath:   e.deps.StatePath,
			CatalogPath: e.deps.CatalogPath,
			ArchivesDir: e.deps.ArchivesDir,
			GitHub:      e.deps.GitHub,
		}
		res, err := doctor.Run(params)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				em.Send(CanceledEvent{OpID: opID})
				return
			}
			em.Send(FailedEvent{OpID: opID, Err: err})
			return
		}
		for _, c := range res.Checks {
			em.Send(DoctorSectionEvent{
				OpID:    opID,
				Section: c.Section,
				Title:   c.Title,
				Status:  translateSeverity(c.Severity),
				Detail:  c.Detail,
				Hint:    c.Hint,
				Action:  doctorActionFor(c),
			})
		}
		em.Send(CompletedEvent{OpID: opID, Kind: CompletedSuccess})
	}()

	return OpHandle{
		Events:  em.Events(),
		Resolve: func(token int64, choice ConflictChoice) { em.pending.resolve(token, choice) },
	}
}

// translateSeverity maps internal/doctor severity to engine DoctorStatus.
func translateSeverity(s doctor.Severity) DoctorStatus {
	switch s {
	case doctor.SeverityOK:
		return DoctorOK
	case doctor.SeverityWarn:
		return DoctorWarn
	case doctor.SeverityFail:
		return DoctorFail
	default:
		return DoctorSkip
	}
}

// doctorActionFor maps a Check to an actionable remedy, when known.
// Section names are human-readable strings; we match on lowercase substring
// to be robust against minor wording changes ("Catalog cache" → catalog).
func doctorActionFor(c doctor.Check) DoctorAction {
	if c.Severity == doctor.SeverityOK {
		return ActionNone
	}
	section := strings.ToLower(c.Section)
	switch {
	case strings.Contains(section, "catalog"):
		return ActionRefreshCatalog
	case strings.Contains(section, "font cache") || strings.Contains(section, "fc-cache"):
		return ActionRefreshFontCache
	}
	return ActionNone
}
