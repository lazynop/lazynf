package doctor

// Run executes all six diagnostic sections in fixed order and returns the
// aggregated Result. Never returns an error in the current design — every
// finding is reported through Check.Severity, not the function return.
func Run(p Params) (*Result, error) {
	res := &Result{}
	res.Checks = append(res.Checks, checkPaths(p)...)
	res.Checks = append(res.Checks, checkFcCache()...)
	res.Checks = append(res.Checks, checkGitHub(p.GitHub)...)
	res.Checks = append(res.Checks, checkManifest(p.StatePath)...)
	res.Checks = append(res.Checks, checkCatalog(p.CatalogPath)...)
	res.Checks = append(res.Checks, checkOrphans(p.FontDir, p.StatePath, p.CatalogPath)...)
	return res, nil
}
