package build

import "fmt"

var (
	Revision string
	Version  string
	Date     string
)

func String() string {
	return fmt.Sprintf("Version: %s Revision: %s Date: (%s)\n", Version, Revision, Date)
}
