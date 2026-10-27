package policycheck

import (
	"strings"
	"testing"

	_ "github.com/aipay-dev/protobuf/gen/go/aipay/approval/v1"
	authzv1 "github.com/aipay-dev/protobuf/gen/go/aipay/authz/v1"
	_ "github.com/aipay-dev/protobuf/gen/go/aipay/common/v1"
	_ "github.com/aipay-dev/protobuf/gen/go/aipay/events/v1"
	_ "github.com/aipay-dev/protobuf/gen/go/aipay/iam/v1"
	_ "github.com/aipay-dev/protobuf/gen/go/aipay/ledger/v1"
	_ "github.com/aipay-dev/protobuf/gen/go/aipay/payment/v1"
	_ "github.com/aipay-dev/protobuf/gen/go/aipay/reconciliation/v1"
	_ "github.com/aipay-dev/protobuf/gen/go/aipay/settlement/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	none     = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED
	required = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED
	mfa      = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT
	single   = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_SINGLE_USE
)

func image(policy *authzv1.MethodPolicy) *descriptorpb.FileDescriptorSet {
	method := &descriptorpb.MethodDescriptorProto{Name: proto.String("Do")}
	if policy != nil {
		method.Options = &descriptorpb.MethodOptions{}
		proto.SetExtension(method.Options, authzv1.E_Policy, policy)
	}
	return &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name: proto.String("aipay/test/v1/test.proto"), Package: proto.String("aipay.test.v1"),
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("TestService"), Method: []*descriptorpb.MethodDescriptorProto{method}}},
	}}}
}

func rule(assertion authzv1.AssertionRequirement, maxAge uint32, permissions []string, callers ...string) *authzv1.CallerRule {
	return &authzv1.CallerRule{Callers: callers, Assertion: assertion, Permissions: permissions, RecentAuthMaxAgeSeconds: maxAge}
}

func TestPolicyRules(t *testing.T) {
	withdraw := []string{"settlement.withdrawal.create"}
	cases := map[string]struct {
		policy *authzv1.MethodPolicy
		want   string
	}{
		"missing":             {nil, "missing (aipay.authz.v1.policy)"},
		"empty":               {&authzv1.MethodPolicy{}, "no caller rules"},
		"no callers":          {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(none, 0, nil)}}, "no callers"},
		"bad caller":          {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(none, 0, nil, "Backend")}}, "invalid caller"},
		"duplicate caller":    {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(none, 0, nil, "settlement"), rule(required, 0, withdraw, "settlement")}}, "more than one rule"},
		"unspecified":         {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_UNSPECIFIED, 0, nil, "backend")}}, "must be declared"},
		"unknown enum":        {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(9, 0, nil, "backend")}}, "must be declared"},
		"service with perm":   {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(none, 0, withdraw, "backend")}}, "permissions require an assertion"},
		"required no perm":    {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(required, 0, nil, "backend")}}, "needs acceptable permissions"},
		"wildcard perm":       {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(required, 0, []string{"payment.*"}, "backend")}}, "invalid permission"},
		"duplicate perm":      {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(required, 0, []string{"a.b", "a.b"}, "backend")}}, "duplicate permission"},
		"mfa without max age": {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(mfa, 0, withdraw, "backend")}}, "between 1 and 3600"},
		"single max age big":  {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(single, 3601, withdraw, "backend")}}, "between 1 and 3600"},
		"max age without mfa": {&authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule(required, 300, withdraw, "backend")}}, "requires MFA_RECENT"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			problems := Check(image(tc.policy))
			if len(problems) == 0 || !strings.Contains(problems[0].Error(), tc.want) {
				t.Fatalf("problems=%v, want %q", problems, tc.want)
			}
		})
	}
	valid := &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{
		rule(single, 300, withdraw, "backend"),
		rule(none, 0, nil, "ledger", "reconciliation"),
	}}
	if problems := Check(image(valid)); len(problems) != 0 {
		t.Fatal(problems)
	}
}

func TestEmptyImageRejected(t *testing.T) {
	if problems := Check(&descriptorpb.FileDescriptorSet{}); len(problems) == 0 {
		t.Fatal("an image without AI Pay RPCs passed")
	}
}

// The generated SDK must satisfy the same check as the source image.
func TestGeneratedContractsDeclarePolicies(t *testing.T) {
	set := &descriptorpb.FileDescriptorSet{}
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if strings.HasPrefix(string(file.Package()), "aipay.") {
			set.File = append(set.File, protodesc.ToFileDescriptorProto(file))
		}
		return true
	})
	if problems := Check(set); len(problems) != 0 {
		t.Fatal(problems)
	}
}
