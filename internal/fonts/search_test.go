package fonts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearch_CaseInsensitiveSubstring(t *testing.T) {
	all := []string{"0xProto", "FiraCode", "FiraMono", "JetBrainsMono", "Hack"}

	assert.Equal(t, []string{"FiraMono", "JetBrainsMono"}, Search(all, "mono"))
	assert.Equal(t, []string{"FiraMono", "JetBrainsMono"}, Search(all, "MONO"))
	assert.Equal(t, []string{"FiraCode", "FiraMono"}, Search(all, "Fira"))
	assert.Equal(t, []string{"0xProto"}, Search(all, "proto"))
}

func TestSearch_EmptyQuery_ReturnsAll(t *testing.T) {
	all := []string{"A", "B", "C"}
	assert.Equal(t, all, Search(all, ""))
}

func TestSearch_NoMatches_ReturnsEmpty(t *testing.T) {
	all := []string{"A", "B"}
	assert.Empty(t, Search(all, "zzz"))
}

func TestSuggest_FindsClosestSingleMatch(t *testing.T) {
	all := []string{"FiraCode", "FiraMono", "JetBrainsMono", "Hack"}
	// Single best suggestion for a near-miss
	got := Suggest(all, "FiraCod", 1)
	assert.Equal(t, []string{"FiraCode"}, got)
}

func TestSuggest_LimitsResults(t *testing.T) {
	all := []string{"FiraCode", "FiraMono", "FireFire"}
	got := Suggest(all, "Fir", 2)
	assert.Len(t, got, 2)
}
