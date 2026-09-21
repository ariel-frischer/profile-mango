package install

import (
	"fmt"
	"sort"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

func ApplyPlan(plan Plan, options ApplyOptions) (ApplyReport, error) {
	if options.ExpectedPlanID == "" {
		return ApplyReport{}, fmt.Errorf("apply requires an expected plan ID")
	}
	if options.ExpectedPlanID != plan.PlanID {
		return ApplyReport{}, fmt.Errorf("expected plan %s does not match actual plan %s", options.ExpectedPlanID, plan.PlanID)
	}
	if plan.Status != StatusReady && plan.Status != StatusNoop {
		return ApplyReport{}, fmt.Errorf("install plan is not applicable: %s", plan.Status)
	}
	if err := validateSources(plan); err != nil {
		return ApplyReport{}, err
	}
	for _, target := range plan.Targets {
		if len(target.checks) == 0 {
			continue
		}
		if err := installfs.Preflight(target.checks); err != nil {
			return ApplyReport{}, fmt.Errorf("preflight %s: %w", target.Target.String(), err)
		}
	}
	report := ApplyReport{Status: "committed", Targets: make([]ApplyTargetResult, 0, len(plan.Targets))}
	for _, target := range sortedTargetPlans(plan.Targets) {
		result := ApplyTargetResult{Target: target.Target.String(), Status: target.Status}
		if len(target.changes) > 0 {
			applied, err := installfs.Apply(target.changes, installfs.ApplyOptions{PlanID: plan.PlanID, Backup: plan.Backup})
			result.Status = applied.Status
			if err != nil {
				result.Error = err.Error()
				report.Status = "partial"
				report.Targets = append(report.Targets, result)
				return report, fmt.Errorf("apply %s: %w", result.Target, err)
			}
		}
		report.Targets = append(report.Targets, result)
	}
	return report, nil
}

func sortedTargetPlans(targets []TargetPlan) []TargetPlan {
	result := append([]TargetPlan(nil), targets...)
	sort.Slice(result, func(i, j int) bool { return result[i].Target.String() < result[j].Target.String() })
	return result
}
