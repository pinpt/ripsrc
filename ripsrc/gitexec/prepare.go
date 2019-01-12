package gitexec

import (
	"context"
	"fmt"
	"os"
)

/*
const gitattributesConfig = `
** -binary
** -text
** diff
`
*/
func Prepare(ctx context.Context, gitCommand, repoDir string) error {
	headCommit := headCommit(ctx, gitCommand, repoDir)
	if headCommit == "" {
		return fmt.Errorf("can't get head commit for repo: %v", repoDir)
	}

	/*
		// assuming regular checkout
		gitDir := filepath.Join(repoDir, ".git")
		dotGit, err := dirExists(gitDir)
		if err != nil {
			return err
		}
		if !dotGit {
			// it's a bare repo
			gitDir = repoDir
		}

		err = ioutil.WriteFile(filepath.Join(gitDir, "info", "attributes"), []byte(gitattributesConfig), 0666)
		if err != nil {
			return err
		}
	*/

	return nil
}

func dirExists(dir string) (bool, error) {
	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, fmt.Errorf("can't check if dir exists dir: %v err: %v", dir, err)
		}
	}
	if !stat.IsDir() {
		return false, nil
	}
	return true, nil
}
