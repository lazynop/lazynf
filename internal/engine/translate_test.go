package engine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazynf/internal/fonts"
)

// collectFunc captures the events translateXxxEvent produces so we can assert
// on the concrete types without spinning up a real op channel.
func collectFunc() (func(Event), *[]Event) {
	out := &[]Event{}
	return func(ev Event) { *out = append(*out, ev) }, out
}

func TestTranslateInstallEvent_AllKinds(t *testing.T) {
	cases := []struct {
		name string
		in   fonts.Event
		want func(*testing.T, []Event)
	}{
		{
			name: "download-start",
			in:   fonts.Event{Font: "X", Kind: fonts.EventDownloadStart},
			want: func(t *testing.T, got []Event) {
				require.Len(t, got, 1)
				le := got[0].(LogEvent)
				require.Equal(t, "downloading", le.Message)
			},
		},
		{
			name: "download-done-silent",
			in:   fonts.Event{Font: "X", Kind: fonts.EventDownloadDone},
			want: func(t *testing.T, got []Event) {
				require.Empty(t, got)
			},
		},
		{
			name: "extract-start",
			in:   fonts.Event{Font: "X", Kind: fonts.EventExtractStart},
			want: func(t *testing.T, got []Event) {
				require.Equal(t, "extracting", got[0].(LogEvent).Message)
			},
		},
		{
			name: "extract-done",
			in:   fonts.Event{Font: "X", Kind: fonts.EventExtractDone},
			want: func(t *testing.T, got []Event) {
				require.Equal(t, "extracted", got[0].(LogEvent).Message)
			},
		},
		{
			name: "cache-refresh",
			in:   fonts.Event{Kind: fonts.EventCacheRefresh},
			want: func(t *testing.T, got []Event) {
				require.Equal(t, KindFcCache, got[0].(StartedEvent).Kind)
			},
		},
		{
			name: "install-success",
			in:   fonts.Event{Font: "X", Kind: fonts.EventInstallSuccess},
			want: func(t *testing.T, got []Event) {
				ce := got[0].(CompletedEvent)
				require.Equal(t, CompletedSuccess, ce.Kind)
				require.Equal(t, "installed", ce.Detail)
			},
		},
		{
			name: "install-skipped",
			in:   fonts.Event{Font: "X", Kind: fonts.EventInstallSkipped},
			want: func(t *testing.T, got []Event) {
				ce := got[0].(CompletedEvent)
				require.Equal(t, CompletedSkipped, ce.Kind)
				require.Equal(t, "already installed", ce.Detail)
			},
		},
		{
			name: "install-error",
			in:   fonts.Event{Font: "X", Kind: fonts.EventInstallError, Err: errors.New("boom")},
			want: func(t *testing.T, got []Event) {
				fe := got[0].(FailedEvent)
				require.EqualError(t, fe.Err, "boom")
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			send, got := collectFunc()
			translateInstallEvent(1, c.in, send)
			c.want(t, *got)
		})
	}
}

func TestTranslateUpdateEvent_AllKinds(t *testing.T) {
	cases := []struct {
		name string
		in   fonts.Event
		want func(*testing.T, []Event)
	}{
		{
			name: "download-start",
			in:   fonts.Event{Font: "X", Kind: fonts.EventDownloadStart},
			want: func(t *testing.T, got []Event) {
				require.Equal(t, "downloading", got[0].(LogEvent).Message)
			},
		},
		{
			name: "extract-start",
			in:   fonts.Event{Font: "X", Kind: fonts.EventExtractStart},
			want: func(t *testing.T, got []Event) {
				require.Equal(t, "extracting", got[0].(LogEvent).Message)
			},
		},
		{
			name: "extract-done",
			in:   fonts.Event{Font: "X", Kind: fonts.EventExtractDone},
			want: func(t *testing.T, got []Event) {
				require.Equal(t, "extracted", got[0].(LogEvent).Message)
			},
		},
		{
			name: "cache-refresh",
			in:   fonts.Event{Kind: fonts.EventCacheRefresh},
			want: func(t *testing.T, got []Event) {
				require.Equal(t, KindFcCache, got[0].(StartedEvent).Kind)
			},
		},
		{
			name: "install-success-as-updated",
			in:   fonts.Event{Font: "X", Kind: fonts.EventInstallSuccess},
			want: func(t *testing.T, got []Event) {
				ce := got[0].(CompletedEvent)
				require.Equal(t, "updated", ce.Detail)
			},
		},
		{
			name: "install-skipped-as-fresh",
			in:   fonts.Event{Font: "X", Kind: fonts.EventInstallSkipped},
			want: func(t *testing.T, got []Event) {
				ce := got[0].(CompletedEvent)
				require.Equal(t, "already fresh", ce.Detail)
			},
		},
		{
			name: "install-error",
			in:   fonts.Event{Font: "X", Kind: fonts.EventInstallError, Err: errors.New("nope")},
			want: func(t *testing.T, got []Event) {
				require.EqualError(t, got[0].(FailedEvent).Err, "nope")
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			send, got := collectFunc()
			translateUpdateEvent(1, c.in, send)
			c.want(t, *got)
		})
	}
}

func TestTranslateImportEvent_AllKinds(t *testing.T) {
	t.Run("import-start", func(t *testing.T) {
		send, got := collectFunc()
		translateImportEvent(1, fonts.Event{Font: "X", Kind: fonts.EventImportStart}, send, map[string]struct{}{})
		require.Equal(t, "importing", (*got)[0].(LogEvent).Message)
	})
	t.Run("import-success", func(t *testing.T) {
		send, got := collectFunc()
		translateImportEvent(1, fonts.Event{Font: "X", Kind: fonts.EventImportSuccess}, send, map[string]struct{}{})
		ce := (*got)[0].(CompletedEvent)
		require.Equal(t, CompletedSuccess, ce.Kind)
		require.Equal(t, "imported", ce.Detail)
	})
	t.Run("import-skipped", func(t *testing.T) {
		send, got := collectFunc()
		translateImportEvent(1, fonts.Event{Font: "X", Kind: fonts.EventImportSkipped}, send, map[string]struct{}{})
		ce := (*got)[0].(CompletedEvent)
		require.Equal(t, CompletedSkipped, ce.Kind)
		require.Equal(t, "already imported", ce.Detail)
	})
	t.Run("import-error-recorded", func(t *testing.T) {
		send, got := collectFunc()
		emitted := map[string]struct{}{}
		translateImportEvent(1, fonts.Event{Font: "X", Kind: fonts.EventImportError, Err: errors.New("e")}, send, emitted)
		require.EqualError(t, (*got)[0].(FailedEvent).Err, "e")
		_, ok := emitted["X"]
		require.True(t, ok, "import error target should be recorded in emitted set")
	})
}
