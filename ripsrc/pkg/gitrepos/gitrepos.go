package gitrepos

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func IterDir(dir string, maxRecursion int, cb func(repo string) error) error {
	stat, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("can't stat passed dir, err: %v", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("passed dir is a file, expecting a dir")
	}

	// check if contains .git
	containsDotGit, err := dirContainsDir(dir, ".git")
	if err != nil {
		return err
	}

	if containsDotGit {
		err := cb(dir)
		if err != nil {
			return err
		}
		return nil
	}

	loc, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("can't convert passed dir to absolute path, err: %v", err)
	}

	if filepath.Ext(loc) == ".git" {
		containsObjects, err := dirContainsDir(dir, "objects")
		if err != nil {
			return err
		}
		if containsObjects {
			err := cb(dir)
			if err != nil {
				return err
			}
		}
	}

	if maxRecursion == 0 {
		return nil
	}

	subs, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("can't read passed dir, err: %v", err)
	}

	for _, sub := range subs {
		if !sub.IsDir() {
			continue
		}
		err := IterDir(filepath.Join(dir, sub.Name()), maxRecursion-1, cb)
		if err != nil {
			return err
		}
	}

	return nil
}

func dirContainsDir(dir string, sub string) (bool, error) {
	stat, err := os.Stat(filepath.Join(dir, sub))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, fmt.Errorf("can't check if dir contains %v, dir: %v err: %v", sub, dir, err)
		}
	}
	if !stat.IsDir() {
		return false, nil
	}

	return true, nil
}
