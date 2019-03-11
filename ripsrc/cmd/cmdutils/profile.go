package cmdutils

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/profile"
)

func EnableProfiling(kind string) (onEnd func()) {
	dir, _ := ioutil.TempDir("", "ripsrc-profile")

	var stop func()

	onEnd = func() {
		stop()
		fn := filepath.Join(dir, kind+".pprof")
		fmt.Printf("to view profile, run `go tool pprof --pdf %s`\n", fn)
	}

	switch kind {
	case "cpu":
		{
			stop = profile.Start(profile.CPUProfile, profile.ProfilePath(dir), profile.Quiet).Stop
		}
	case "mem":
		{
			stop = profile.Start(profile.MemProfile, profile.ProfilePath(dir), profile.Quiet).Stop
		}
	case "trace":
		{
			stop = profile.Start(profile.TraceProfile, profile.ProfilePath(dir), profile.Quiet).Stop
		}
	case "block":
		{
			stop = profile.Start(profile.BlockProfile, profile.ProfilePath(dir), profile.Quiet).Stop
		}
	case "mutex":
		{
			stop = profile.Start(profile.MutexProfile, profile.ProfilePath(dir), profile.Quiet).Stop
		}
	default:
		{
			panic("unexpected profile: " + kind)
		}
	}
	return
}
