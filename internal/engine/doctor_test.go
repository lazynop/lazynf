package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazynf/internal/github"
	"github.com/stretchr/testify/require"
)

func TestRunDoctor_EmitsSectionsAndCompleted(t *testing.T) {
	dir := t.TempDir()
	e := New(Deps{
		FontDir:     filepath.Join(dir, "fonts"),
		StatePath:   filepath.Join(dir, "state.json"),
		CatalogPath: filepath.Join(dir, "catalog.json"),
		GitHub:      github.NewClient(),
	})
	handle := e.RunDoctor(context.Background())
	events := DrainEvents(t, handle)

	var (
		started   []StartedEvent
		sections  []DoctorSectionEvent
		completed []CompletedEvent
		failed    []FailedEvent
	)
	for _, ev := range events {
		switch x := ev.(type) {
		case StartedEvent:
			started = append(started, x)
		case DoctorSectionEvent:
			sections = append(sections, x)
		case CompletedEvent:
			completed = append(completed, x)
		case FailedEvent:
			failed = append(failed, x)
		}
	}
	require.NotEmpty(t, started)
	require.Equal(t, "doctor", started[0].Kind)
	require.NotEmpty(t, sections, "expected at least one DoctorSectionEvent")
	require.Len(t, completed, 1)
	require.Empty(t, failed)
}

func TestRunDoctor_SeverityTranslation(t *testing.T) {
	require := require.New(t)
	// Cover all branches of translateSeverity (uses doctor's exported constants).
	require.Equal(DoctorOK, translateSeverity(0))   // SeverityOK
	require.Equal(DoctorWarn, translateSeverity(1)) // SeverityWarn
	require.Equal(DoctorFail, translateSeverity(2)) // SeverityFail
	require.Equal(DoctorSkip, translateSeverity(99))
}
