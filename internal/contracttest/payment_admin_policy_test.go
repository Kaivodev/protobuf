package contracttest

import (
	"slices"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Pin the entire management surface and its trust boundary: backend needs a
// user proof, while only reconciliation can retrieve independent ruling proof.
func TestPaymentAdminMethodAuthorization(t *testing.T) {
	type expected struct {
		caller     string
		assertion  authzv1.AssertionRequirement
		permission string
	}
	required := authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED
	methods := map[protoreflect.Name]expected{}
	for name, permission := range map[protoreflect.Name]string{
		"SubmitChannelAccount":  "payment.channel.manage",
		"ApproveChannelAccount": "payment.channel.approve",
		"DisableChannelAccount": "payment.channel.disable",
		"SubmitPaymentMethod":   "payment.method.manage",
		"ApprovePaymentMethod":  "payment.method.approve",
		"SubmitRiskPolicy":      "payment.risk.submit",
		"ApproveRiskPolicy":     "payment.risk.approve",
		"SavePricingDraft":      "payment.pricing.manage",
		"SubmitPricingVersion":  "payment.pricing.manage",
		"ApprovePricingVersion": "payment.pricing.approve",
		"UploadPlugin":          "payment.plugin.upload",
		"ApprovePlugin":         "payment.plugin.approve",
		"RequestPluginEnable":   "payment.plugin.enable",
		"ApprovePluginEnable":   "payment.plugin.approve",
		"SubmitReversal":        "payment.reversal.submit",
		"ApproveReversal":       "payment.reversal.approve",
	} {
		methods[name] = expected{"backend", required, permission}
	}
	for _, group := range []struct {
		file    protoreflect.FileDescriptor
		name    protoreflect.Name
		methods map[protoreflect.Name]expected
	}{
		{paymentv1.File_aipay_payment_v1_channel_admin_service_proto, "ChannelAdminService", methods},
		{paymentv1.File_aipay_payment_v1_dispute_service_proto, "DisputeService", map[protoreflect.Name]expected{
			"RegisterDispute":      {"backend", required, "payment.dispute.register"},
			"SubmitDisputeRuling":  {"backend", required, "payment.dispute.submit"},
			"ApproveDisputeRuling": {"backend", required, "payment.dispute.approve"},
			"GetDisputeRuling":     {"reconciliation", authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED, ""},
		}},
	} {
		t.Run(string(group.name), func(t *testing.T) {
			service := group.file.Services().ByName(group.name)
			if group.file.Services().Len() != 1 || service == nil || service.Methods().Len() != len(group.methods) {
				t.Fatal("unreviewed management RPC surface")
			}
			for name, want := range group.methods {
				t.Run(string(name), func(t *testing.T) {
					method := service.Methods().ByName(name)
					if method == nil || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
						t.Fatal("missing RPC or authorization policy")
					}
					policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
					if len(policy.GetRules()) != 1 {
						t.Fatalf("unexpected caller rules: %v", policy)
					}
					rule := policy.GetRules()[0]
					var permissions []string
					if want.permission != "" {
						permissions = []string{want.permission}
					}
					if !slices.Equal(rule.GetCallers(), []string{want.caller}) || rule.GetAssertion() != want.assertion || !slices.Equal(rule.GetPermissions(), permissions) || rule.GetRecentAuthMaxAgeSeconds() != 0 {
						t.Fatalf("unexpected authorization: %v", rule)
					}
					if method.IsStreamingClient() || method.IsStreamingServer() {
						t.Fatal("management methods must remain unary")
					}
				})
			}
		})
	}
}

// Prevent a fourth funds direction or a caller-supplied destination/amount from
// silently appearing in ruling input. The read side must retain the complete
// original signed bytes/assertions and evidence, not just a success flag/hash.
func TestPaymentDisputeRulingBoundary(t *testing.T) {
	outcomes := paymentv1.DisputeRulingOutcome(0).Descriptor().Values()
	wantOutcomes := []protoreflect.Name{
		"DISPUTE_RULING_OUTCOME_UNSPECIFIED",
		"DISPUTE_RULING_OUTCOME_RELEASE_TO_ORIGINAL_RECIPIENT",
		"DISPUTE_RULING_OUTCOME_RETURN_TO_ORIGINAL_SELLER",
		"DISPUTE_RULING_OUTCOME_REFUND",
	}
	if outcomes.Len() != len(wantOutcomes) {
		t.Fatal("unreviewed dispute outcome")
	}
	for i, name := range wantOutcomes {
		if outcomes.Get(i).Name() != name || outcomes.Get(i).Number() != protoreflect.EnumNumber(i) {
			t.Fatalf("unexpected outcome at %d: %v", i, outcomes.Get(i))
		}
	}
	request := (&paymentv1.SubmitDisputeRulingRequest{}).ProtoReflect().Descriptor()
	wantFields := []protoreflect.Name{"idempotency_key", "dispute_id", "expected_control_version", "outcome", "reason"}
	if request.Fields().Len() != len(wantFields) {
		t.Fatal("unreviewed ruling input: must not accept account/amount/snapshot overrides")
	}
	for i, name := range wantFields {
		if request.Fields().Get(i).Name() != name || request.Fields().Get(i).Number() != protoreflect.FieldNumber(i+1) {
			t.Fatalf("unexpected ruling field at %d", i)
		}
	}
	response := (&paymentv1.GetDisputeRulingResponse{}).ProtoReflect().Descriptor()
	for name, typ := range map[protoreflect.Name]protoreflect.FullName{
		"approval": "aipay.payment.v1.PaymentApproval",
		"evidence": "aipay.payment.v1.DisputeEvidence",
	} {
		field := response.Fields().ByName(name)
		if field == nil || field.Kind() != protoreflect.MessageKind || !field.HasPresence() || field.Message().FullName() != typ {
			t.Fatalf("missing independent ruling proof: %s", name)
		}
	}
	// The approval wrapper must keep the reusable approval.v1 wire proof, which
	// already carries raw request_bytes/digest and original WebAuthn assertions.
	approval := (&paymentv1.PaymentApproval{}).ProtoReflect().Descriptor()
	for name, typ := range map[protoreflect.Name]protoreflect.FullName{
		"approval_request": "aipay.approval.v1.ApprovalRequest",
		"approvals":        "aipay.approval.v1.Approval",
		"material":         "aipay.payment.v1.PaymentApprovalMaterial",
	} {
		field := approval.Fields().ByName(name)
		if field == nil || field.Kind() != protoreflect.MessageKind || field.Message().FullName() != typ {
			t.Fatalf("missing signed material/approval records: %s", name)
		}
	}
}
