package shared

import "strings"

type Issue struct {
	Key      string              `json:"issue"`
	Reason   string              `json:"reason"`
	FollowUp string              `json:"followUp"`
	Samples  map[string][]string `json:"samples"`
}
type CheckFunc func(Entry, *Context) error
type Matcher func(Entry, Sample, Issue, error) bool

// RunExpectedFailure checks every sample, rejects missing/duplicate evidence,
// changed failures and partial fixes, and never swallows infrastructure errors.
func RunExpectedFailure(e Entry, c *Context, level string, issue *Issue, check CheckFunc, match Matcher) (bool, error) {
	if issue == nil || len(issue.Samples[level]) == 0 {
		return false, check(e, c)
	}
	names := map[string]bool{}
	for _, name := range issue.Samples[level] {
		names[name] = true
		n := 0
		for _, s := range c.Samples {
			if s.Name == name {
				n++
			}
		}
		if n != 1 {
			return false, failure("expected-failure fixture missing or duplicated: " + name)
		}
	}
	for _, s := range c.Samples {
		parts := strings.Split(e.Path, ".")
		hasParent := len(parts) == 1 || len(Values(s.Report, strings.Join(parts[:len(parts)-1], "."))) > 0
		if e.Kind == "property" && (!e.Required || !hasParent) && !names[s.Name] && len(Values(s.Report, e.Path)) == 0 {
			continue
		}
		one := c.With(s)
		err := check(e, &one)
		c.Warnings = append(c.Warnings, one.Warnings[len(c.Warnings):]...)
		if !names[s.Name] {
			if err != nil {
				return false, err
			}
			continue
		}
		if err == nil {
			return false, failure("unexpected pass: " + e.ID + " / " + s.Name)
		}
		if !match(e, s, *issue, err) {
			return false, err
		}
	}
	return true, nil
}
