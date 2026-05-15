// Package app houses the root tea.Model. layout.go computes pane sizes from
// terminal dimensions.
package app

// Layout describes the rectangle each pane occupies.
type Layout struct {
	ListW, ListH     int
	DetailW, DetailH int
	LogW, LogH       int
	StatusW          int
	Vertical         bool
}

// Compute returns pane sizes for a terminal of width w and height h.
// The log pane is hidden if showLog == false (its rows fold back into list and
// detail). A width below 80 cols flips to a vertical layout (list above detail).
func Compute(w, h int, showLog bool) Layout {
	if w < 1 || h < 1 {
		return Layout{}
	}
	statusH := 1
	logH := 0
	if showLog {
		logH = min(8, h/3)
	}
	bodyH := max(1, h-statusH-logH)
	if w >= 80 {
		listW := max(20, w*38/100)
		return Layout{
			ListW:   listW,
			ListH:   bodyH,
			DetailW: w - listW,
			DetailH: bodyH,
			LogW:    w,
			LogH:    logH,
			StatusW: w,
		}
	}
	listH := bodyH / 2
	return Layout{
		ListW:    w,
		ListH:    listH,
		DetailW:  w,
		DetailH:  bodyH - listH,
		LogW:     w,
		LogH:     logH,
		StatusW:  w,
		Vertical: true,
	}
}
