package agentcheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	DefaultManifestPath = "docs/dev/agents/sources.json"
	DefaultTimeout      = 10 * time.Second
	DefaultMaxBodyBytes = 4 << 20
	DefaultMaxRedirects = 5
)

type Status string

const (
	StatusUnchanged   Status = "unchanged"
	StatusChanged     Status = "changed"
	StatusUnversioned Status = "unversioned"
	StatusUnavailable Status = "unavailable"
	StatusRelocated   Status = "relocated"
	StatusNotChecked  Status = "not_checked"
)

type Manifest struct {
	SchemaVersion int      `json:"schema_version"`
	RetrievedAt   string   `json:"retrieved_at"`
	Purpose       string   `json:"purpose"`
	Targets       []Target `json:"targets"`
}

type Target struct {
	ID                   string                     `json:"id"`
	Name                 string                     `json:"name"`
	Reference            string                     `json:"reference"`
	ProductStatus        string                     `json:"product_status"`
	DocumentationBasis   string                     `json:"documentation_basis"`
	InstalledObservation string                     `json:"installed_observation"`
	TestedEvidence       string                     `json:"tested_evidence"`
	SupportedCapability  string                     `json:"supported_capability"`
	VersionContext       map[string]json.RawMessage `json:"version_context"`
	Sources              []Source                   `json:"sources"`
}

type Source struct {
	URL     string `json:"url,omitempty"`
	Locator string `json:"locator,omitempty"`
	Kind    string `json:"kind"`
	SHA256  string `json:"sha256,omitempty"`
}

type Options struct {
	Client       *http.Client
	TargetID     string
	Timeout      time.Duration
	MaxBodyBytes int64
	MaxRedirects int
	OnProgress   func(Progress)
}

type Progress struct {
	Completed int
	Total     int
	TargetID  string
	State     Status
}

type Report struct {
	SchemaVersion int            `json:"schema_version"`
	ManifestPath  string         `json:"manifest"`
	RetrievedAt   string         `json:"retrieved_at"`
	Targets       []TargetReport `json:"targets"`
	Summary       Summary        `json:"summary"`
}

type TargetReport struct {
	ID             string                     `json:"id"`
	Name           string                     `json:"name"`
	ProductStatus  string                     `json:"product_status"`
	VersionContext map[string]json.RawMessage `json:"version_context"`
	Sources        []SourceReport             `json:"sources"`
}

type SourceReport struct {
	URL            string   `json:"url,omitempty"`
	Locator        string   `json:"locator,omitempty"`
	Kind           string   `json:"kind"`
	ChangedArea    string   `json:"changed_area"`
	State          Status   `json:"state"`
	HTTPStatus     int      `json:"http_status,omitempty"`
	FinalURL       string   `json:"final_url,omitempty"`
	Redirects      []string `json:"redirects,omitempty"`
	ExpectedSHA256 string   `json:"expected_sha256,omitempty"`
	ObservedSHA256 string   `json:"observed_sha256,omitempty"`
	Error          string   `json:"error,omitempty"`
	Recommendation string   `json:"recommendation"`
}

type Summary struct {
	Unchanged   int `json:"unchanged"`
	Changed     int `json:"changed"`
	Unversioned int `json:"unversioned"`
	Unavailable int `json:"unavailable"`
	Relocated   int `json:"relocated"`
	NotChecked  int `json:"not_checked"`
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("reading source manifest %q: %w", path, err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decoding source manifest %q: %w", path, err)
	}
	if err := manifest.validate(); err != nil {
		return Manifest{}, fmt.Errorf("validating source manifest %q: %w", path, err)
	}
	return manifest, nil
}

func CheckFile(ctx context.Context, path string, options Options) (Report, error) {
	manifest, err := LoadManifest(path)
	if err != nil {
		return Report{}, err
	}

	report, err := Check(ctx, manifest, options)
	if err != nil {
		return Report{}, err
	}
	report.ManifestPath = path
	return report, nil
}

func Check(ctx context.Context, manifest Manifest, options Options) (Report, error) {
	if err := manifest.validate(); err != nil {
		return Report{}, fmt.Errorf("validating source manifest: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	options = options.withDefaults()
	targets, err := selectTargets(manifest.Targets, options.TargetID)
	if err != nil {
		return Report{}, err
	}
	client := configuredClient(options)
	total := 0
	for _, target := range targets {
		total += len(target.Sources)
	}
	completed := 0
	report := Report{
		SchemaVersion: manifest.SchemaVersion,
		RetrievedAt:   manifest.RetrievedAt,
		Targets:       make([]TargetReport, 0, len(targets)),
	}
	for _, target := range targets {
		targetReport := checkTarget(ctx, client, target, options, &completed, total)
		report.Targets = append(report.Targets, targetReport)
	}
	report.Summary = summarize(report.Targets)
	return report, nil
}

func (o Options) withDefaults() Options {
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.MaxBodyBytes <= 0 {
		o.MaxBodyBytes = DefaultMaxBodyBytes
	}
	if o.MaxRedirects <= 0 {
		o.MaxRedirects = DefaultMaxRedirects
	}
	return o
}

func configuredClient(options Options) *http.Client {
	if options.Client == nil {
		return &http.Client{Timeout: options.Timeout}
	}
	client := *options.Client
	if client.Timeout <= 0 {
		client.Timeout = options.Timeout
	}
	return &client
}

func selectTargets(targets []Target, targetID string) ([]Target, error) {
	selected := append([]Target(nil), targets...)
	if targetID != "" {
		selected = selected[:0]
		for _, target := range targets {
			if target.ID == targetID {
				selected = append(selected, target)
			}
		}
		if len(selected) == 0 {
			return nil, fmt.Errorf("target %q not found in source manifest", targetID)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].ID < selected[j].ID
	})
	return selected, nil
}

func checkTarget(ctx context.Context, client *http.Client, target Target, options Options, completed *int, total int) TargetReport {
	sources := append([]Source(nil), target.Sources...)
	sort.SliceStable(sources, func(i, j int) bool {
		return sourceKey(sources[i]) < sourceKey(sources[j])
	})
	report := TargetReport{
		ID:             target.ID,
		Name:           target.Name,
		ProductStatus:  target.ProductStatus,
		VersionContext: target.VersionContext,
		Sources:        make([]SourceReport, 0, len(sources)),
	}
	for _, source := range sources {
		if options.OnProgress != nil {
			options.OnProgress(Progress{Completed: *completed, Total: total, TargetID: target.ID})
		}
		result := checkSource(ctx, client, source, options)
		report.Sources = append(report.Sources, result)
		*completed++
		if options.OnProgress != nil {
			options.OnProgress(Progress{Completed: *completed, Total: total, TargetID: target.ID, State: result.State})
		}
	}
	return report
}

func sourceKey(source Source) string {
	if source.URL != "" {
		return source.URL
	}
	return source.Locator
}

func checkSource(ctx context.Context, client *http.Client, source Source, options Options) SourceReport {
	report := SourceReport{
		URL:            source.URL,
		Locator:        source.Locator,
		Kind:           source.Kind,
		ChangedArea:    sourceCategory(source.Kind),
		ExpectedSHA256: source.SHA256,
	}
	if source.URL == "" {
		return finish(report, StatusNotChecked, "non-public locator was not fetched")
	}
	if err := validateHTTPURL(source.URL); err != nil {
		return finish(report, StatusNotChecked, err.Error())
	}

	fetched := fetch(ctx, client, source.URL, options)
	report.HTTPStatus = fetched.status
	report.FinalURL = fetched.finalURL
	report.Redirects = fetched.redirects
	if fetched.err != nil {
		return finish(report, StatusUnavailable, fetched.err.Error())
	}
	if fetched.status < http.StatusOK || fetched.status >= http.StatusMultipleChoices {
		if len(fetched.redirects) > 0 && fetched.status >= http.StatusMultipleChoices && fetched.status < http.StatusBadRequest {
			return finish(report, StatusRelocated, redirectMessage(fetched))
		}
		return finish(report, StatusUnavailable, fmt.Sprintf("source returned HTTP %d", fetched.status))
	}
	report.ObservedSHA256 = fetched.sha256
	if len(fetched.redirects) > 0 {
		return finish(report, StatusRelocated, redirectMessage(fetched))
	}
	if source.SHA256 == "" {
		return finish(report, StatusUnversioned, "source is reachable but has no manifest hash")
	}
	if strings.EqualFold(source.SHA256, fetched.sha256) {
		return finish(report, StatusUnchanged, "")
	}
	return finish(report, StatusChanged, "response hash differs from the manifest")
}

type fetchResult struct {
	status    int
	finalURL  string
	redirects []string
	sha256    string
	err       error
}

func fetch(ctx context.Context, baseClient *http.Client, sourceURL string, options Options) fetchResult {
	redirects := make([]string, 0)
	client := *baseClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		redirects = append(redirects, req.URL.String())
		if len(via) >= options.MaxRedirects {
			return http.ErrUseLastResponse
		}
		return nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return fetchResult{redirects: redirects, err: fmt.Errorf("creating request: %w", err)}
	}
	response, err := client.Do(request)
	if err != nil {
		return fetchResult{redirects: redirects, err: fmt.Errorf("fetching source: %w", err)}
	}
	defer func() { _ = response.Body.Close() }()
	finalURL := ""
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL.String()
	}
	result := fetchResult{
		status:    response.StatusCode,
		finalURL:  finalURL,
		redirects: redirects,
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return result
	}
	result.sha256, result.err = hashBody(response.Body, options.MaxBodyBytes)
	if result.err != nil {
		result.err = fmt.Errorf("reading response body: %w", result.err)
	}
	return result
}

func hashBody(body io.Reader, maxBytes int64) (string, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxBytes {
		return "", fmt.Errorf("response body exceeds %d-byte limit", maxBytes)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func finish(report SourceReport, state Status, detail string) SourceReport {
	report.State = state
	report.Error = detail
	report.Recommendation = recommendation(state)
	return report
}

func redirectMessage(result fetchResult) string {
	if result.finalURL == "" {
		return "source redirected"
	}
	return fmt.Sprintf("source redirected to %s", result.finalURL)
}

func recommendation(state Status) string {
	switch state {
	case StatusUnchanged:
		return "No source refresh is indicated; this does not establish target compatibility."
	case StatusChanged:
		return "Review the source diff and run targeted native probes; content drift is not a compatibility regression or support promotion."
	case StatusUnversioned:
		return "Use reachability and version context only; record a reviewed revision or hash and run targeted native probes before compatibility claims."
	case StatusUnavailable:
		return "Retry or inspect the source manually; unavailable is not unchanged and does not justify support changes."
	case StatusRelocated:
		return "Review the redirect and source diff, retain prior provenance, and run targeted native probes before updating references."
	case StatusNotChecked:
		return "No fetch was attempted because the locator is non-public or uses an unsupported scheme."
	default:
		return "Review this source state before changing references or support records."
	}
}

func summarize(targets []TargetReport) Summary {
	var summary Summary
	for _, target := range targets {
		for _, source := range target.Sources {
			switch source.State {
			case StatusUnchanged:
				summary.Unchanged++
			case StatusChanged:
				summary.Changed++
			case StatusUnversioned:
				summary.Unversioned++
			case StatusUnavailable:
				summary.Unavailable++
			case StatusRelocated:
				summary.Relocated++
			case StatusNotChecked:
				summary.NotChecked++
			}
		}
	}
	return summary
}

func validateHTTPURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("source URL is invalid and was not fetched: %w", err)
	}
	if parsed.User != nil {
		return fmt.Errorf("source URL contains user info and was not fetched")
	}
	if parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("source URL is not a public HTTP(S) URL and was not fetched")
	}
	return nil
}

func sourceCategory(kind string) string {
	lower := strings.ToLower(kind)
	switch {
	case strings.Contains(lower, "schema"):
		return "schema"
	case strings.Contains(lower, "documentation"):
		return "documentation"
	case strings.Contains(lower, "release"):
		return "release context"
	case strings.Contains(lower, "snapshot"):
		return "local snapshot"
	default:
		return kind
	}
}

func (manifest Manifest) validate() error {
	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema_version %d", manifest.SchemaVersion)
	}
	if len(manifest.Targets) == 0 {
		return fmt.Errorf("manifest has no targets")
	}
	seen := make(map[string]struct{}, len(manifest.Targets))
	for _, target := range manifest.Targets {
		if target.ID == "" {
			return fmt.Errorf("target has empty id")
		}
		if _, exists := seen[target.ID]; exists {
			return fmt.Errorf("duplicate target %q", target.ID)
		}
		seen[target.ID] = struct{}{}
		if err := validateSources(target); err != nil {
			return err
		}
	}
	return nil
}

func validateSources(target Target) error {
	seen := make(map[string]struct{}, len(target.Sources))
	for _, source := range target.Sources {
		key := sourceKey(source)
		if key == "" || (source.URL != "" && source.Locator != "") {
			return fmt.Errorf("target %q has a source with an invalid locator", target.ID)
		}
		if source.Kind == "" {
			return fmt.Errorf("target %q source %q has empty kind", target.ID, key)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("target %q has duplicate source %q", target.ID, key)
		}
		seen[key] = struct{}{}
		if source.SHA256 != "" {
			if len(source.SHA256) != sha256.Size*2 {
				return fmt.Errorf("target %q source %q has invalid sha256", target.ID, key)
			}
			if _, err := hex.DecodeString(source.SHA256); err != nil {
				return fmt.Errorf("target %q source %q has invalid sha256: %w", target.ID, key, err)
			}
		}
	}
	return nil
}

func FormatText(w io.Writer, report Report) error {
	return FormatTextStyled(w, report, nil)
}

func FormatTextStyled(w io.Writer, report Report, formatStatus func(Status) string) error {
	if _, err := fmt.Fprintf(w, "manifest: %s\nretrieved_at: %s\n", report.ManifestPath, report.RetrievedAt); err != nil {
		return err
	}
	for _, target := range report.Targets {
		if err := formatTarget(w, target, formatStatus); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "summary: unchanged=%d changed=%d unversioned=%d unavailable=%d relocated=%d not_checked=%d\n", report.Summary.Unchanged, report.Summary.Changed, report.Summary.Unversioned, report.Summary.Unavailable, report.Summary.Relocated, report.Summary.NotChecked)
	return err
}

func formatTarget(w io.Writer, target TargetReport, formatStatus func(Status) string) error {
	if _, err := fmt.Fprintf(w, "target: %s (%s) product_status=%s version_context=%s\n", target.ID, target.Name, target.ProductStatus, formatContext(target.VersionContext)); err != nil {
		return err
	}
	for _, source := range target.Sources {
		location := source.URL
		if location == "" {
			location = source.Locator
		}
		state := string(source.State)
		if formatStatus != nil {
			state = formatStatus(source.State)
		}
		if _, err := fmt.Fprintf(w, "  source: %s kind=%s category=%s state=%s\n", location, source.Kind, source.ChangedArea, state); err != nil {
			return err
		}
		if err := formatSourceDetails(w, source); err != nil {
			return err
		}
	}
	return nil
}

func formatSourceDetails(w io.Writer, source SourceReport) error {
	if source.ObservedSHA256 != "" {
		if _, err := fmt.Fprintf(w, "    observed_sha256: %s\n", source.ObservedSHA256); err != nil {
			return err
		}
	}
	if len(source.Redirects) > 0 {
		if _, err := fmt.Fprintf(w, "    redirects: %s\n", strings.Join(source.Redirects, " -> ")); err != nil {
			return err
		}
	}
	if source.Error != "" {
		if _, err := fmt.Fprintf(w, "    detail: %s\n", source.Error); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "    recommendation: %s\n", source.Recommendation)
	return err
}

func formatContext(values map[string]json.RawMessage) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.TrimSpace(string(values[key]))))
	}
	return strings.Join(parts, " ")
}

func FormatJSON(w io.Writer, report Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
