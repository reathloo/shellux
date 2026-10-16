package systeminfo

import (
	"runtime"
	"slices"
	"strings"
	"time"
)

// LoggedUsers counts distinct user names in local login sessions, not processes.
type LoggedUsers struct {
	Names     []string
	Available bool
}

func LoggedUsersInfo() LoggedUsers {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return LoggedUsers{}
	}
	sample := detailSampleWithin(1200*time.Millisecond, "who")
	if sample.err != nil {
		return LoggedUsers{}
	}
	return parseLoggedUsers(sample.output)
}

func parseLoggedUsers(output string) LoggedUsers {
	seen := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || len(fields[0]) > 64 {
			continue
		}
		seen[fields[0]] = true
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	slices.Sort(names)
	return LoggedUsers{Names: names, Available: true}
}
