package build

import "fmt"

var (
	// Revison returns the revision
	Revision string

	// Version returns the version
	Version string

	// Date returns the date
	Date string
)

// String returns the full string representation of version information
func String() string {
	return fmt.Sprintf("Version: %s Revision: %s Date: (%s)\n", Version, Revision, Date)
}
