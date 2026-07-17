package placement

import (
	"github.com/Omotolani98/monocron/internal/contracts"
)

// SelectRunners returns runners that match all required labels.
func SelectRunners(runners []contracts.Runner, required contracts.Labels) []contracts.Runner {
	if len(required) == 0 {
		return runners
	}
	var out []contracts.Runner
	for _, r := range runners {
		if matches(r.Labels, required) {
			out = append(out, r)
		}
	}
	return out
}

func matches(labels, required contracts.Labels) bool {
	for k, v := range required {
		if labels[k] != v {
			return false
		}
	}
	return true
}
