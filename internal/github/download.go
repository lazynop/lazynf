package github

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// ProgressFunc is called as bytes are written to disk.
// `written` is total bytes written so far; `total` is the expected size in
// bytes (from Content-Length, 0 if unknown).
type ProgressFunc func(written, total int64)

// DownloadAsset GETs url and writes the body to destPath.
// On HTTP error, no partial file is left at destPath.
//
// We use a plain http.Client (not the GitHub-API-flavored one) because release
// asset downloads redirect to S3-like URLs that don't accept the GitHub auth
// header verbatim. For MVP, asset URLs are public.
func DownloadAsset(url, destPath string, onProgress ProgressFunc) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download %s: status %d: %s", url, resp.StatusCode, string(body))
	}

	var total int64
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		total, _ = strconv.ParseInt(cl, 10, 64)
	}

	tmp := destPath + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create dest %s: %w", tmp, err)
	}

	pr := &progressReader{r: resp.Body, total: total, onProgress: onProgress}
	if _, err := io.Copy(out, pr); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, destPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename to %s: %w", destPath, err)
	}
	return nil
}

type progressReader struct {
	r          io.Reader
	written    int64
	total      int64
	onProgress ProgressFunc
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	if n > 0 {
		pr.written += int64(n)
		if pr.onProgress != nil {
			pr.onProgress(pr.written, pr.total)
		}
	}
	return n, err
}
