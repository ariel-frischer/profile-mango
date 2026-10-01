package install

import (
	"fmt"
	"sort"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

func ApplyPlan(plan Plan, options ApplyOptions) (ApplyReport, error) {
	if options.ExpectedPlanID == "" {
		err := fmt.Errorf("apply requires an expected plan ID")
		return preflightApplyReport(plan, err, ""), err
	}
	if options.ExpectedPlanID != plan.PlanID {
		err := fmt.Errorf("expected plan %s does not match actual plan %s", options.ExpectedPlanID, plan.PlanID)
		return preflightApplyReport(plan, err, ""), err
	}
	if plan.Status != StatusReady && plan.Status != StatusNoop {
		err := fmt.Errorf("install plan is not applicable: %s", plan.Status)
		return preflightApplyReport(plan, err, ""), err
	}
	if err := validateSources(plan); err != nil {
		return preflightApplyReport(plan, err, ""), err
	}
	for _, target := range sortedTargetPlans(plan.Targets) {
		if len(target.checks) == 0 {
			continue
		}
		if err := installfs.Preflight(target.checks); err != nil {
			cause := fmt.Errorf("preflight %s: %w", target.Target.String(), err)
			return preflightApplyReport(plan, cause, target.Target.String()), cause
		}
	}
	return applyTargetChanges(plan)
}

func preflightApplyReport(plan Plan, cause error, failedTarget string) ApplyReport {
	targets := sortedTargetPlans(plan.Targets)
	report := ApplyReport{Status: StatusNotAttempted, Targets: make([]ApplyTargetResult, 0, len(targets))}
	hasPending := false
	for _, target := range targets {
		result := ApplyTargetResult{Target: target.installedTarget().String(), Status: target.Status}
		changed := len(target.changes) > 0 || target.Status == StatusReady
		if changed {
			result.Status = StatusNotAttempted
			hasPending = true
		}
		if cause != nil && (changed || target.Target.String() == failedTarget) {
			result.Error = cause.Error()
		}
		report.Targets = append(report.Targets, result)
	}
	if len(report.Targets) == 0 {
		report.Targets = append(report.Targets, ApplyTargetResult{Target: "install", Status: StatusNotAttempted, Error: cause.Error()})
	} else if cause != nil && !hasPending && failedTarget == "" {
		report.Targets[0].Error = cause.Error()
	}
	return report
}

func applyTargetChanges(plan Plan) (ApplyReport, error) {
	targets := sortedTargetPlans(plan.Targets)
	var changes []installfs.Change
	var anchors []string
	for _, target := range targets {
		changes = append(changes, target.changes...)
		anchors = append(anchors, target.ConfigPath)
	}
	applied, err := installfs.Apply(changes, installfs.ApplyOptions{PlanID: plan.PlanID, Backup: plan.Backup, Anchors: anchors})
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
	if applied.Status == "committed" {
		if err := writeJournalRefs(targets, plan.PlanID, applied); err != nil {
			return report, fmt.Errorf("install committed, but undo cannot locate its journal: %w", err)
		}
	}
	return report, nil
}

func targetApplyReport(targets []TargetPlan, status string, cause error) ApplyReport {
	report := ApplyReport{Status: status, Targets: make([]ApplyTargetResult, 0, len(targets))}
	for _, target := range targets {
		result := ApplyTargetResult{Target: target.installedTarget().String(), Status: target.Status}
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
