package placement

import (
	"testing"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/google/uuid"
)

func TestSelectRunners(t *testing.T) {
	runners := []contracts.Runner{
		{ID: uuid.New(), Labels: contracts.Labels{"zone": "home", "os": "linux"}},
		{ID: uuid.New(), Labels: contracts.Labels{"zone": "cloud", "os": "linux"}},
		{ID: uuid.New(), Labels: contracts.Labels{"zone": "home", "os": "darwin"}},
	}

	matched := SelectRunners(runners, contracts.Labels{"zone": "home"})
	if len(matched) != 2 {
		t.Fatalf("expected 2 matched runners, got %d", len(matched))
	}

	matched = SelectRunners(runners, contracts.Labels{"zone": "home", "os": "linux"})
	if len(matched) != 1 {
		t.Fatalf("expected 1 matched runner, got %d", len(matched))
	}

	matched = SelectRunners(runners, contracts.Labels{})
	if len(matched) != 3 {
		t.Fatalf("expected all runners, got %d", len(matched))
	}

	matched = SelectRunners(runners, contracts.Labels{"gpu": "true"})
	if len(matched) != 0 {
		t.Fatalf("expected 0 matched runners, got %d", len(matched))
	}
}
