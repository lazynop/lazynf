package fontcache

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeRefresher_RecordsCall(t *testing.T) {
	f := &FakeRefresher{}
	require.NoError(t, f.Refresh(context.Background()))
	assert.True(t, f.Called)
}

func TestFakeRefresher_PropagatesError(t *testing.T) {
	want := errors.New("boom")
	f := &FakeRefresher{Err: want}
	err := f.Refresh(context.Background())
	assert.True(t, errors.Is(err, want))
	assert.True(t, f.Called)
}

func TestDefault_NotNil(t *testing.T) {
	assert.NotNil(t, Default())
}
