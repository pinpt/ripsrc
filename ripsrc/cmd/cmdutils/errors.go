package cmdutils

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

func ExitWithErr(err error) {
	fmt.Fprintln(color.Output, color.RedString("failed with error: %v\n", err.Error()))
}

func ExitWithErrs(errs []error) {
	if len(errs) == 0 {
		panic("no errors")
	}
	if len(errs) == 1 {
		ExitWithErr(errs[0])
		return
	}
	for _, err := range errs {
		fmt.Fprintln(color.Output, color.RedString("%v\n", err))
	}
	fmt.Fprintln(color.Output, color.RedString("failed"))
	os.Exit(1)
}
