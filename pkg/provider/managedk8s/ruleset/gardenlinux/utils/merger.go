// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/gardener/diki/pkg/rule"
	"github.com/gardener/diki/pkg/ruleset"
)

// MergeRulesetResults merges multiple ruleset.RulesetResult entries into a single one.
// Assumes all inputs are from the same ruleset (same ID, name, version) and contain
// the same set of rules (same count and matching IDs). The merged result concatenates
// the CheckResults of corresponding rules in input order.
func MergeRulesetResults(results []ruleset.RulesetResult) ruleset.RulesetResult {
	if len(results) == 0 {
		return ruleset.RulesetResult{}
	}

	if len(results) == 1 {
		return results[0]
	}

	base := results[0]

	mergedByID := make(map[string]rule.RuleResult, len(base.RuleResults))

	for _, rr := range base.RuleResults {
		mergedByID[rr.RuleID] = rule.RuleResult{
			RuleID:       rr.RuleID,
			RuleName:     rr.RuleName,
			Severity:     rr.Severity,
			CheckResults: append([]rule.CheckResult(nil), rr.CheckResults...),
		}
	}

	for i := 1; i < len(results); i++ {
		r := results[i]
		for _, rr := range r.RuleResults {
			merged, _ := mergedByID[rr.RuleID]
			merged.CheckResults = append(merged.CheckResults, rr.CheckResults...)
			mergedByID[rr.RuleID] = merged
		}
	}

	out := ruleset.RulesetResult{
		RulesetID:      base.RulesetID,
		RulesetName:    base.RulesetName,
		RulesetVersion: base.RulesetVersion,
		RuleResults:    make([]rule.RuleResult, 0, len(base.RuleResults)),
	}

	for _, rr := range base.RuleResults {
		out.RuleResults = append(out.RuleResults, mergedByID[rr.RuleID])
	}

	return out
}
