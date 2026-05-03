package archive

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFonts_FiltersToTTFAndOTF(t *testing.T) {
	dest := t.TempDir()
	files, err := ExtractFonts("testdata/sample.zip", dest)
	require.NoError(t, err)

	sort.Strings(files)
	assert.Equal(t, []string{
		"JetBrainsMonoNerdFont-Bold.ttf",
		"JetBrainsMonoNerdFont-Regular.ttf",
		"JetBrainsMonoNerdFontMono-Italic.otf",
	}, files)

	// README, LICENSE, cheatsheet should NOT be on disk.
	for _, name := range []string{"README.md", "LICENSE", "nested"} {
		_, err := os.Stat(filepath.Join(dest, name))
		assert.True(t, os.IsNotExist(err), "should not extract %s", name)
	}
}

func TestExtractFonts_FilesArePlacedFlatInDest(t *testing.T) {
	dest := t.TempDir()
	_, err := ExtractFonts("testdata/sample.zip", dest)
	require.NoError(t, err)

	// Even if a font were nested in the zip, it should be placed flat in dest.
	// sample.zip's font entries are at the root, but the nested cheatsheet
	// must not leak through.
	entries, err := os.ReadDir(dest)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, e.IsDir(), "no subdirectories should be created in dest")
	}
}

func TestExtractFonts_ContentMatchesArchive(t *testing.T) {
	dest := t.TempDir()
	_, err := ExtractFonts("testdata/sample.zip", dest)
	require.NoError(t, err)

	cases := map[string][]byte{
		"JetBrainsMonoNerdFont-Regular.ttf":    []byte("FAKE_TTF_BYTES_REGULAR"),
		"JetBrainsMonoNerdFont-Bold.ttf":       []byte("FAKE_TTF_BYTES_BOLD"),
		"JetBrainsMonoNerdFontMono-Italic.otf": []byte("FAKE_OTF_BYTES"),
	}
	for name, want := range cases {
		got, err := os.ReadFile(filepath.Join(dest, name))
		require.NoError(t, err, name)
		assert.Equal(t, want, got, name)
	}
}

func TestExtractFonts_MissingArchive_ReturnsError(t *testing.T) {
	_, err := ExtractFonts("testdata/does_not_exist.zip", t.TempDir())
	assert.Error(t, err)
}

func TestIsFontFile(t *testing.T) {
	assert.True(t, isFontFile("foo.ttf"))
	assert.True(t, isFontFile("foo.TTF"))
	assert.True(t, isFontFile("foo.otf"))
	assert.True(t, isFontFile("foo.OTF"))
	assert.False(t, isFontFile("foo.txt"))
	assert.False(t, isFontFile("README.md"))
	assert.False(t, isFontFile("foo"))
}
