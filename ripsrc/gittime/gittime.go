package gittime

import "time"

// Parse parses timestamp in default git format.
func Parse(d string) (time.Time, error) {
	//Tue Nov 27 21:55:36 2018 +0100
	return time.Parse("Mon Jan 2 15:04:05 2006 -0700", d)
}
