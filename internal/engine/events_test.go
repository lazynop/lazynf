package engine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventInterface_AllVariantsImplemented(t *testing.T) {
	var events = []Event{
		StartedEvent{OpID: 1},
		ProgressEvent{OpID: 1},
		LogEvent{OpID: 1, Level: LevelInfo, Message: "hi"},
		CompletedEvent{OpID: 1, Kind: CompletedSuccess},
		FailedEvent{OpID: 1, Err: errors.New("boom")},
		CanceledEvent{OpID: 1},
		ConflictEvent{OpID: 1, Token: 42},
		DoctorSectionEvent{OpID: 1, Section: "paths", Status: DoctorOK},
	}
	for _, ev := range events {
		require.Equal(t, OpID(1), ev.GetOpID())
	}
}

func TestEngine_NextOpID_Monotone(t *testing.T) {
	e := New(Deps{})
	a := e.nextOpID()
	b := e.nextOpID()
	require.Greater(t, b, a)
}

func TestEngine_NextToken_Monotone(t *testing.T) {
	e := New(Deps{})
	a := e.nextToken()
	b := e.nextToken()
	require.Greater(t, b, a)
}
