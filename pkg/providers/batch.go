package providers

// batchWithFallback calls cmdFn with all names at once.
// If the batch call fails and there are multiple names, it retries each name
// individually to identify which ones actually failed.
// Returns nil if all succeed, or a map of item name → error for failures.
// Missing entries in the returned map mean success.
func batchWithFallback(names []string, cmdFn func([]string) error) map[string]error {
	if len(names) == 0 {
		return nil
	}
	batchErr := cmdFn(names)
	if batchErr == nil {
		return nil
	}
	if len(names) == 1 {
		return map[string]error{names[0]: batchErr}
	}
	// Batch failed — retry individually to surface specific failures.
	failed := make(map[string]error)
	for _, name := range names {
		if err := cmdFn([]string{name}); err != nil {
			failed[name] = err
		}
	}
	return failed
}
