package threshold

// MinComments returns the minimum required number of review comments
// based on how many files were changed in the PR.
func MinComments(changedFiles int) int {
	switch {
	case changedFiles <= 4:
		return 1
	case changedFiles <= 20:
		return 3
	default:
		return 5
	}
}
