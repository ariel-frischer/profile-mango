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
	return applyTargetChanges(plan)
}

func applyTargetChanges(plan Plan) (ApplyReport, error) {
	targets := sortedTargetPlans(plan.Targets)
	var changes []installfs.Change
	for _, target := range targets {
		changes = append(changes, target.changes...)
	}
	applied, err := installfs.Apply(changes, installfs.ApplyOptions{PlanID: plan.PlanID, Backup: plan.Backup})
	status := applied.Status
	if status == "noop" {
		status = "committed"
	}
	if err != nil && status == "" {
		status = "failed"
	}
	report := targetApplyReport(targets, status, err)
	if err != nil {
		return report, fmt.Errorf("apply installation transaction: %w", err)
	}
	return report, nil
}

func targetApplyReport(targets []TargetPlan, status string, cause error) ApplyReport {
	report := ApplyReport{Status: status, Targets: make([]ApplyTargetResult, 0, len(targets))}
	for _, target := range targets {
		result := ApplyTargetResult{Target: target.Target.String(), Status: target.Status}
		if len(target.changes) > 0 {
			result.Status = status
			if cause != nil {
				result.Error = cause.Error()
			}
		}
		report.Targets = append(report.Targets, result)
	}
	return report
}

func sortedTargetPlans(targets []TargetPlan) []TargetPlan {
	result := append([]TargetPlan(nil), targets...)
	sort.Slice(result, func(i, j int) bool { return result[i].Target.String() < result[j].Target.String() })
	return result
}
