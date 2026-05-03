// Package archive extracts font files from Nerd Fonts release zips.
//
// Nerd Fonts archives contain .ttf/.otf files plus README, LICENSE, glyph
// cheatsheets, and other metadata. We only want the font files themselves,
// flat in the destination directory.
package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractFonts opens the zip at archivePath and extracts every .ttf/.otf entry
// into destDir (flat layout — basename only). Returns the list of basenames
// extracted, in extraction order.
//
// Non-font entries (README, LICENSE, cheatsheets, nested directories) are
// silently skipped. The destination dir must already exist.
func ExtractFonts(archivePath, destDir string) ([]string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive %s: %w", archivePath, err)
	}
	defer r.Close()

	var extracted []string
	for _, f := range r.File {
		base := filepath.Base(f.Name)
		if !isFontFile(base) {
			continue
		}
		if err := writeOne(f, filepath.Join(destDir, base)); err != nil {
			return extracted, fmt.Errorf("extract %s: %w", base, err)
		}
		extracted = append(extracted, base)
	}
	return extracted, nil
}

func isFontFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".ttf" || ext == ".otf"
}

func writeOne(f *zip.File, destPath string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}
