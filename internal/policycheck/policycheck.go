// Package policycheck verifies that every Kaivo RPC declares a well-formed
// (aipay.authz.v1.policy) method option. It runs from `make check` against the
// image built from proto/, so a method without a policy fails the build.
package policycheck

import (
	"fmt"
	"regexp"
	"strings"

	authzv1 "github.com/aipay-dev/protobuf/gen/go/aipay/authz/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// MaxRecentAuthSeconds bounds recent_auth_max_age_seconds so a typo cannot
// silently turn "recent" into "any time".
const MaxRecentAuthSeconds = 3600

var (
	servicePattern    = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	permissionPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
)

// Check returns one error per invalid or missing policy in Kaivo packages.
func Check(set *descriptorpb.FileDescriptorSet) []error {
	var problems []error
	methods := 0
	for _, file := range set.GetFile() {
		if file.GetPackage() != "aipay" && !strings.HasPrefix(file.GetPackage(), "aipay.") {
			continue
		}
		for _, service := range file.GetService() {
			for _, method := range service.GetMethod() {
				methods++
				name := fmt.Sprintf("%s.%s/%s", file.GetPackage(), service.GetName(), method.GetName())
				for _, problem := range checkMethod(method) {
					problems = append(problems, fmt.Errorf("%s (%s): %s", name, file.GetName(), problem))
				}
			}
		}
	}
	if methods == 0 {
		problems = append(problems, fmt.Errorf("no Kaivo RPC methods found; the descriptor image is incomplete"))
	}
	return problems
}

func checkMethod(method *descriptorpb.MethodDescriptorProto) []string {
	options := method.GetOptions()
	if options == nil || !proto.HasExtension(options, authzv1.E_Policy) {
		return []string{"missing (aipay.authz.v1.policy)"}
	}
	policy, ok := proto.GetExtension(options, authzv1.E_Policy).(*authzv1.MethodPolicy)
	if !ok || len(policy.GetRules()) == 0 {
		return []string{"policy has no caller rules"}
	}
	var problems []string
	seen := map[string]bool{}
	for i, rule := range policy.GetRules() {
		report := func(format string, args ...any) {
			problems = append(problems, fmt.Sprintf("rule %d: ", i+1)+fmt.Sprintf(format, args...))
		}
		if len(rule.GetCallers()) == 0 {
			report("no callers")
		}
		for _, caller := range rule.GetCallers() {
			if !servicePattern.MatchString(caller) {
				report("invalid caller %q", caller)
			} else if seen[caller] {
				report("caller %q appears in more than one rule", caller)
			}
			seen[caller] = true
		}
		permissions := map[string]bool{}
		for _, permission := range rule.GetPermissions() {
			if !permissionPattern.MatchString(permission) {
				report("invalid permission %q", permission)
			} else if permissions[permission] {
				report("duplicate permission %q", permission)
			}
			permissions[permission] = true
		}
		maxAge := rule.GetRecentAuthMaxAgeSeconds()
		switch rule.GetAssertion() {
		case authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED:
			if len(rule.GetPermissions()) != 0 {
				report("permissions require an assertion")
			}
			if maxAge != 0 {
				report("recent_auth_max_age_seconds requires MFA_RECENT or SINGLE_USE")
			}
		case authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED:
			if len(rule.GetPermissions()) == 0 {
				report("an assertion requirement needs acceptable permissions")
			}
			if maxAge != 0 {
				report("recent_auth_max_age_seconds requires MFA_RECENT or SINGLE_USE")
			}
		case authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT, authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_SINGLE_USE:
			if len(rule.GetPermissions()) == 0 {
				report("an assertion requirement needs acceptable permissions")
			}
			if maxAge == 0 || maxAge > MaxRecentAuthSeconds {
				report("recent_auth_max_age_seconds must be between 1 and %d", MaxRecentAuthSeconds)
			}
		default:
			report("assertion requirement must be declared")
		}
		if len(rule.ProtoReflect().GetUnknown()) != 0 {
			report("unknown policy fields")
		}
	}
	if len(policy.ProtoReflect().GetUnknown()) != 0 {
		problems = append(problems, "unknown policy fields")
	}
	return problems
}
