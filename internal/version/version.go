package version

import "fmt"

const Name = "shellux"

// Value is replaced by GoReleaser from the Git tag. Local builds keep the
// development baseline so --version remains useful outside a release.
var Value = "0.1.2"

func String() string {
	return fmt.Sprintf("%s v%s", Name, Value)
}
