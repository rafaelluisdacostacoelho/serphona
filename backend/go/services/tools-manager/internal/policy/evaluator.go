package policy

// Decision represents the outcome of a policy evaluation.
type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
)

// String returns the string representation.
func (d Decision) String() string { return string(d) }

// Rule captures a single allow/deny statement.
type Rule struct {
	Effect   Decision // allow or deny
	TenantID string   // optional tenant scoping; empty means any
	AgentID  string   // optional agent scoping
	Scopes   []string // required scopes, all must be present if set
	Roles    []string // required roles, any match
	ToolID   string   // optional tool scoping
	Env      string   // environment scoping, e.g., "dev", "prod"
	Weight   int      // precedence; higher weight wins among same effect; deny beats allow on tie
}

// Request carries attributes to evaluate.
type Request struct {
	TenantID string
	AgentID  string
	Scopes   []string
	Roles    []string
	ToolID   string
	Env      string
}

// Evaluator evaluates policy rules with deterministic precedence.
type Evaluator struct {
	rules []Rule
}

// NewEvaluator constructs an evaluator from a static ruleset.
func NewEvaluator(rules []Rule) *Evaluator {
	return &Evaluator{rules: rules}
}

// Evaluate returns allow/deny given the request and ruleset.
// Precedence: (1) exact matches with highest weight; (2) deny beats allow when weights tie; (3) fallback allow when no rules match.
func (e *Evaluator) Evaluate(req Request) Decision {
	matched := []Rule{}
	for _, r := range e.rules {
		if !matchTenant(r, req) {
			continue
		}
		if !matchOptional(r.AgentID, req.AgentID) {
			continue
		}
		if !matchOptional(r.ToolID, req.ToolID) {
			continue
		}
		if !matchOptional(r.Env, req.Env) {
			continue
		}
		if !containsAll(req.Scopes, r.Scopes) {
			continue
		}
		if !containsAny(req.Roles, r.Roles) {
			continue
		}
		matched = append(matched, r)
	}

	if len(matched) == 0 {
		return DecisionAllow
	}

	best := matched[0]
	for _, r := range matched[1:] {
		if r.Weight > best.Weight {
			best = r
			continue
		}
		if r.Weight == best.Weight {
			if r.Effect == DecisionDeny && best.Effect == DecisionAllow {
				best = r
			}
		}
	}

	return best.Effect
}

func matchTenant(r Rule, req Request) bool {
	if r.TenantID == "" {
		return true
	}
	return r.TenantID == req.TenantID
}

func matchOptional(expected, actual string) bool {
	if expected == "" {
		return true
	}
	return expected == actual
}

func containsAll(have, need []string) bool {
	if len(need) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, h := range have {
		set[h] = struct{}{}
	}
	for _, n := range need {
		if _, ok := set[n]; !ok {
			return false
		}
	}
	return true
}

func containsAny(have, options []string) bool {
	if len(options) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, h := range have {
		set[h] = struct{}{}
	}
	for _, opt := range options {
		if _, ok := set[opt]; ok {
			return true
		}
	}
	return false
}
