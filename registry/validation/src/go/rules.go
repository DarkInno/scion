package validation

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

// emailRule validates an address using net/mail.ParseAddress rather than a
// regular expression, as required by the security specification.
type emailRule struct{}

func (emailRule) Validate(value string, present bool) error {
	if !present || value == "" {
		return nil
	}
	addr, err := mail.ParseAddress(value)
	if err != nil {
		return fmt.Errorf("must be a valid email address")
	}
	// Reject the "Display Name <addr@x>" form: the parsed address must equal
	// the input (case-insensitively) so that only a bare address is accepted.
	// This also blocks header-injection style payloads smuggled via the name.
	if !strings.EqualFold(addr.Address, value) {
		return fmt.Errorf("must be a valid email address")
	}
	return nil
}
func (emailRule) Name() string { return "email" }

// urlRule validates that the value is an http or https URL with a host.
type urlRule struct{}

func (urlRule) Validate(value string, present bool) error {
	if !present || value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || u == nil {
		return fmt.Errorf("must be a valid URL")
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("must be a valid URL")
	}
	// Restrict to http/https to prevent javascript:, file:, data: schemes.
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("must be an http or https URL")
	}
}
func (urlRule) Name() string { return "url" }

// uuidRule validates a canonical hyphenated UUID (any version).
type uuidRule struct{}

// uuidRegex is compiled once at package initialization. The pattern is a fixed
// constant and safe to compile with MustCompile.
var uuidRegex = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

func (uuidRule) Validate(value string, present bool) error {
	if !present || value == "" {
		return nil
	}
	if !uuidRegex.MatchString(value) {
		return fmt.Errorf("must be a valid UUID")
	}
	return nil
}
func (uuidRule) Name() string { return "uuid" }

// ipRule validates an IPv4 or IPv6 address using net.ParseIP.
type ipRule struct{}

func (ipRule) Validate(value string, present bool) error {
	if !present || value == "" {
		return nil
	}
	if net.ParseIP(value) == nil {
		return fmt.Errorf("must be a valid IP address")
	}
	return nil
}
func (ipRule) Name() string { return "ip" }

// inRule validates that the value is one of an allowed set.
type inRule struct {
	values []string
	set    map[string]struct{}
}

func newInRule(values []string) inRule {
	set := make(map[string]struct{}, len(values))
	cp := make([]string, len(values))
	for i, v := range values {
		cp[i] = v
		set[v] = struct{}{}
	}
	return inRule{values: cp, set: set}
}

func (r inRule) Validate(value string, present bool) error {
	if !present || value == "" {
		return nil
	}
	if _, ok := r.set[value]; !ok {
		return fmt.Errorf("must be one of: %s", strings.Join(r.values, ", "))
	}
	return nil
}
func (inRule) Name() string { return "in" }

// regexRule validates that the value matches a precompiled regular expression.
//
// The pattern is compiled exactly once (when the rule is created via
// newRegexRule) and its length is bounded by Options.MaxRegexLength. Go's
// regexp engine is RE2, which guarantees linear-time matching and cannot
// suffer catastrophic backtracking. The length cap plus RE2's linear bound
// together satisfy the "regex compile/match timeout" safety requirement
// without leaking goroutines.
type regexRule struct {
	pattern    string
	re         *regexp.Regexp
	compileErr error
}

func newRegexRule(pattern string, maxLen int) regexRule {
	r := regexRule{pattern: pattern}
	if len(pattern) > maxLen {
		r.compileErr = fmt.Errorf("regex pattern exceeds maximum length of %d characters", maxLen)
		return r
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		r.compileErr = err
		return r
	}
	r.re = re
	return r
}

func (r regexRule) Validate(value string, present bool) error {
	if r.compileErr != nil {
		return fmt.Errorf("regex pattern invalid: %v", r.compileErr)
	}
	if !present || value == "" {
		return nil
	}
	if !r.re.MatchString(value) {
		return fmt.Errorf("has an invalid format")
	}
	return nil
}
func (r regexRule) Name() string { return "regex" }

// customRule delegates validation to a user-supplied function. The function is
// only invoked when the field is present and non-empty, matching the behavior
// of the other format rules.
type customRule struct {
	name string
	fn   func(value string) error
}

func (r customRule) Validate(value string, present bool) error {
	if !present || value == "" {
		return nil
	}
	return r.fn(value)
}
func (r customRule) Name() string { return r.name }
