package main

import (
	"strings"
	"testing"

	"github.com/dictyBase/fp-go-loom/pipelinecheck"
)

// TestPipelineStyle enforces the house fp-go pipeline rules on the
// whole module: handoff wrappers, applied seeds, raw TryCatch, safe
// Printf, redundant folds, point-free branching, fixed-root file
// joins, terminal effects and trivial delegates.
func TestPipelineStyle(t *testing.T) {
	cfg := pipelinecheck.Config{
		Roots: []string{"../.."},
		IsEntrypoint: func(name string) bool {
			return strings.HasPrefix(name, "Run") || strings.HasPrefix(name, "run")
		},
		RequirePointFreeSeed:      true,
		RequirePointFreeBranching: true,
		RequireTrivialDelegate:    true,
		TerminalFunctions:         []string{"PublishStrain", "PublishPlasmid"},
	}
	pipelinecheck.Require(t, cfg)
	pipelinecheck.RequireNoIfErrInTryCatch(t, cfg)
	pipelinecheck.RequireNoNonRawTryCatchCallback(t, cfg)
	pipelinecheck.RequireSafePrintf(t, cfg)
	pipelinecheck.RequireNoRedundantFold(t, cfg)
	pipelinecheck.RequireNoHandRolledFileJoinInArrayMap(t, cfg)
	pipelinecheck.RequireTerminalEffect(t, cfg)
}
