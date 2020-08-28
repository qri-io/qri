package access

import (
	"encoding/json"
	"fmt"
	"strings"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo/profile"
)

// special tokens in access grammer
const (
	matchAll     = "*"
	matchSubject = "_subject"
)

var (
	// ErrAccessDenied is returned by policy enforce
	ErrAccessDenied = fmt.Errorf("access denied")
	log             = golog.Logger("access")
	// DefaultAccessControlPolicyFilename is the file name for the policy
	// expected file is format yaml
	DefaultAccessControlPolicyFilename = "access_control_policy.yaml"
)

// Effect is the set of outcomes a rule can have
type Effect string

const (
	// EffectAllow describes a rule that adds permissions
	EffectAllow = Effect("allow")
	// EffectDeny describes a rule that removes permissions
	EffectDeny = Effect("deny")
)

// Policy is a set of rules
type Policy []Rule

// Rule is a permissions statement. It determines who (subject) can/can't
// (effect) do something (actions) to things (resources)
type Rule struct {
	Title     string    // human-legible title for the rule, informative only
	Subject   string    // User this rule is about
	Resources Resources // Thing being accessed. eg: a dataset,
	Actions   Actions   // Thing user can do
	Effect    Effect    // "allow" or "deny"
}

type rule Rule

// UnmarshalJSON unmarshals the slice of bytes into a Rule
func (r *Rule) UnmarshalJSON(d []byte) error {
	_rule := rule{}
	if err := json.Unmarshal(d, &_rule); err != nil {
		return err
	}

	rule := Rule(_rule)
	if err := rule.Validate(); err != nil {
		return err
	}

	*r = rule
	return nil
}

// Validate returns a descriptive error if the rule is not well-formed
func (r *Rule) Validate() error {
	if r.Subject == "" {
		return fmt.Errorf("rule.Subject is required")
	}
	if r.Effect != EffectAllow && r.Effect != EffectDeny {
		return fmt.Errorf(`rule.Effect must be one of ("allow"|"deny")`)
	}
	if len(r.Resources) == 0 {
		return fmt.Errorf("rule.Resources field is required")
	}
	if len(r.Actions) == 0 {
		return fmt.Errorf("rule.Actions field is required")
	}
	return nil
}

// Enforce evaluates a request against the policy, returning either nil or
// ErrAccessDenied
func (pol Policy) Enforce(subject *profile.Profile, resource, action string) error {
	log.Debugf("policy.Enforce username=%q resource=%q action=%q", subject.Peername, resource, action)
	rsc, err := ParseResource(resource)
	if err != nil {
		return err
	}

	act, err := ParseAction(action)
	if err != nil {
		return err
	}

	for _, rule := range pol {
		log.Debugf("rule=%q effect=%q subject=%t resources=%t actions=%t", rule.Title, rule.Effect,
			(rule.Subject == subject.ID.String() || rule.Subject == matchAll),
			rule.Resources.Contains(rsc, subject.Peername),
			rule.Actions.Contains(act),
		)

		if rule.Effect == EffectAllow &&
			(rule.Subject == subject.ID.String() || rule.Subject == matchAll) &&
			rule.Resources.Contains(rsc, subject.Peername) &&
			rule.Actions.Contains(act) {
			log.Debugf("matched rule title=%q", rule.Title)
			return nil
		}
	}
	return ErrAccessDenied
}

// Resources is a collection of resoureces
type Resources []Resource

// Contains iterates all Resources in the slice, returns true for the first
// resource that contains the given resource
func (rs Resources) Contains(b Resource, subjectUsername string) bool {
	for _, r := range rs {
		if r.Contains(b, subjectUsername) {
			return true
		}
	}
	return false
}

// Resource is a stateful thing in qri
type Resource []string

// MustParseResource wraps ParseResource, panics on error. Useful for tests
func MustParseResource(str string) Resource {
	rsc, err := ParseResource(str)
	if err != nil {
		panic(err)
	}
	return rsc
}

// ParseResource constructs a resource from a string
func ParseResource(str string) (Resource, error) {
	if str == "" {
		return nil, fmt.Errorf("resource string cannot be empty")
	}

	rsc := strings.Split(str, ":")

	foundStar := false
	for _, name := range rsc {
		if name == "*" {
			if foundStar {
				return nil, fmt.Errorf(`invalid resource string %q. '*' character cannot occur twice`, str)
			}
			foundStar = true
		} else if foundStar {
			return nil, fmt.Errorf(`invalid resource string %q. '*' must come last`, str)
		}
	}

	return rsc, nil
}

// MarshalJSON marshals the resource into a string separated by ":"
func (r Resource) MarshalJSON() ([]byte, error) {
	return []byte(strings.Join(r, ":")), nil
}

// UnmarshalJSON unmarshals a slice of bytes into a Resource
func (r *Resource) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	rsc, err := ParseResource(str)
	if err != nil {
		return err
	}

	*r = rsc
	return nil
}

// Contains determins if the subject is referenced in the resource
// returns true if the rule's resource contains the `matchAll` symbol
// and returns true if the rule's resource contains the `matchSubject`
// and the subjectUsername is in the given resource (allows us to create rules
// that say, "only allow subjects to do this action, if the resource matches
// the subject's name"
func (r Resource) Contains(b Resource, subjectUsername string) bool {
	if len(r) > len(b) {
		return false
	}

	for i, aName := range r {
		if aName == matchAll {
			return true
		}
		if aName == matchSubject && b[i] == subjectUsername {
			continue
		}
		if b[i] != aName {
			return false
		}
	}

	return len(r) == len(b)
}

// ResourceStrFromRef takes a dsref.Ref and returns a string that can be parsed
// as a resource
func ResourceStrFromRef(ref dsref.Ref) string {
	return strings.Join([]string{"dataset", ref.Username, ref.Name}, ":")
}

// Actions is a slice of Action
type Actions []Action

// Contains determines if the given action is contained by the Actions
func (as Actions) Contains(b Action) bool {
	for _, a := range as {
		if a.Contains(b) {
			return true
		}
	}
	return false
}

// Action is a description of the action the Subject is attempting to take on
// the Resource
type Action []string

// MustParseAction parses a string into an Action. It panics if the string
// cannot be parsed correctly
func MustParseAction(str string) Action {
	rsc, err := ParseAction(str)
	if err != nil {
		panic(err)
	}
	return rsc
}

// ParseAction parses a string into an Action
func ParseAction(str string) (Action, error) {
	if str == "" {
		return nil, fmt.Errorf("action string cannot be empty")
	}

	rsc := strings.Split(str, ":")

	foundStar := false
	for _, name := range rsc {
		if name == matchAll {
			if foundStar {
				return nil, fmt.Errorf(`invalid action string %q. '*' character cannot occur twice`, str)
			}
			foundStar = true
		} else if foundStar {
			return nil, fmt.Errorf(`invalid action string %q. '*' must come last`, str)
		}
	}

	return rsc, nil
}

// MarshalJSON marshals the Action into a string separated by ":"
func (a Action) MarshalJSON() ([]byte, error) {
	return []byte(strings.Join(a, ":")), nil
}

// UnmarshalJSON unmarshals the given slice of bytes into an Action
func (a *Action) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	act, err := ParseAction(str)
	if err != nil {
		return err
	}

	*a = act
	return nil
}

// Contains determines if the given action is described in the rule's Action
// it returns true if the action matches using the glob `*` pattern
func (a Action) Contains(b Action) bool {
	if len(a) > len(b) {
		return false
	}

	for i, aName := range a {
		if aName == matchAll {
			return true
		}
		if b[i] != aName {
			return false
		}
	}

	return len(a) == len(b)
}
