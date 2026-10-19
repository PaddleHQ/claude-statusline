package gitbranch

var Reconcile = reconcile

// Linked locates dir's .git and reports, as Checkout sees it, whether it belongs to a linked worktree.
func Linked(dir string) (root, main string, ok bool) {
	root, dotGit, info, ok := locate(dir)
	if !ok {
		return "", "", false
	}
	main, ok = linked(info, gitDir(dotGit, info))
	if !ok {
		return "", "", false
	}
	return root, main, true
}
