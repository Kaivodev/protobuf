package contracttest

import (
	"bytes"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	commonv1 "github.com/Kaivodev/protobuf/gen/go/aipay/common/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	reconciliationv1 "github.com/Kaivodev/protobuf/gen/go/aipay/reconciliation/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0230ChannelInstructionLifetime(t *testing.T) {
	field := requireField(t, &paymentv1.ChannelAccountConfiguration{}, "payment_instruction_ttl_seconds", 18, protoreflect.Uint32Kind, "")
	if !field.HasOptionalKeyword() || !field.HasPresence() {
		t.Fatalf("%s requires explicit optional presence", field.FullName())
	}
}

func TestV0230LatePaymentWindowOperation(t *testing.T) {
	field := requireField(t, &reconciliationv1.ReconciliationApprovalMaterial{}, "late_payment_window", 10, protoreflect.MessageKind, "aipay.reconciliation.v1.LatePaymentWindowMaterial")
	if oneof := field.ContainingOneof(); oneof == nil || oneof.Name() != "operation" {
		t.Fatalf("%s must select the operation", field.FullName())
	}
}

func TestV0230PolicyMessageFields(t *testing.T) {
	const (
		u64 = protoreflect.Uint64Kind
		u32 = protoreflect.Uint32Kind
		str = protoreflect.StringKind
		msg = protoreflect.MessageKind

		timestamp              = "google.protobuf.Timestamp"
		assertion              = "aipay.approval.v1.WebAuthnAssertion"
		approvalRequest        = "aipay.approval.v1.ApprovalRequest"
		reconciliationApproval = "aipay.reconciliation.v1.ReconciliationApproval"
		windowVersion          = "aipay.reconciliation.v1.LatePaymentWindowVersion"
		rateVersion            = "aipay.ledger.v1.UsdRatePolicyVersion"
	)
	type field struct {
		number      protoreflect.FieldNumber
		name        protoreflect.Name
		kind        protoreflect.Kind
		messageType protoreflect.FullName
	}
	for _, tc := range []struct {
		message proto.Message
		fields  []field
	}{
		{&reconciliationv1.LatePaymentWindowMaterial{}, []field{
			{1, "version", u64, ""},
			{2, "expected_current_version", u64, ""},
			{3, "window_seconds", u32, ""},
			{4, "reason", str, ""},
		}},
		{&reconciliationv1.LatePaymentWindowVersion{}, []field{
			{1, "version", u64, ""},
			{2, "window_seconds", u32, ""},
			{3, "approval_request_id", str, ""},
			{4, "request_digest", protoreflect.BytesKind, ""},
			{5, "approved_at", msg, timestamp},
		}},
		{&reconciliationv1.SubmitLatePaymentWindowRequest{}, []field{
			{1, "idempotency_key", str, ""},
			{2, "expected_current_version", u64, ""},
			{3, "window_seconds", u32, ""},
			{4, "reason", str, ""},
		}},
		{&reconciliationv1.SubmitLatePaymentWindowResponse{}, []field{
			{1, "approval", msg, reconciliationApproval},
		}},
		{&reconciliationv1.ApproveLatePaymentWindowRequest{}, []field{
			{1, "idempotency_key", str, ""},
			{2, "approval_request_id", str, ""},
			{3, "assertion", msg, assertion},
		}},
		{&reconciliationv1.ApproveLatePaymentWindowResponse{}, []field{
			{1, "approval", msg, reconciliationApproval},
			{2, "version", msg, windowVersion},
		}},
		{&reconciliationv1.GetLatePaymentWindowRequest{}, nil},
		{&reconciliationv1.GetLatePaymentWindowResponse{}, []field{
			{1, "source", protoreflect.EnumKind, ""},
			{2, "window_seconds", u32, ""},
			{3, "current", msg, windowVersion},
			{4, "floor_seconds", u32, ""},
			{5, "cap_seconds", u32, ""},
			{6, "pending", msg, reconciliationApproval},
			{7, "as_of", msg, timestamp},
		}},
		{&reconciliationv1.ListLatePaymentWindowsRequest{}, []field{
			{1, "limit", u32, ""},
			{2, "page_token", str, ""},
		}},
		{&reconciliationv1.ListLatePaymentWindowsResponse{}, []field{
			{1, "versions", msg, windowVersion},
			{2, "next_page_token", str, ""},
			{3, "as_of", msg, timestamp},
		}},
		{&ledgerv1.UsdRatePolicyVersion{}, []field{
			{1, "version", u64, ""},
			{2, "threshold_ppm", u32, ""},
			{3, "approval_request_id", str, ""},
			{4, "approved_at", msg, timestamp},
		}},
		{&ledgerv1.RequestUsdRatePolicyRequest{}, []field{
			{1, "idempotency_key", str, ""},
			{2, "expected_version", u64, ""},
			{3, "threshold_ppm", u32, ""},
			{4, "reason", str, ""},
		}},
		{&ledgerv1.RequestUsdRatePolicyResponse{}, []field{
			{1, "approval_request", msg, approvalRequest},
		}},
		{&ledgerv1.ApproveUsdRatePolicyRequest{}, []field{
			{1, "idempotency_key", str, ""},
			{2, "approval_request_id", str, ""},
			{3, "assertion", msg, assertion},
		}},
		{&ledgerv1.ApproveUsdRatePolicyResponse{}, []field{
			{1, "approval_request", msg, approvalRequest},
			{2, "approvals", u32, ""},
			{3, "version", msg, rateVersion},
		}},
		{&ledgerv1.GetUsdRatePolicyRequest{}, nil},
		{&ledgerv1.GetUsdRatePolicyResponse{}, []field{
			{1, "current", msg, rateVersion},
			{2, "ceiling_ppm", u32, ""},
			{3, "effective_ppm", u32, ""},
			{4, "pending", msg, approvalRequest},
			{5, "as_of", msg, timestamp},
		}},
	} {
		d := tc.message.ProtoReflect().Descriptor()
		t.Run(string(d.Name()), func(t *testing.T) {
			if got := d.Fields().Len(); got != len(tc.fields) {
				t.Fatalf("field count = %d, want %d", got, len(tc.fields))
			}
			for _, want := range tc.fields {
				f := requireField(t, tc.message, want.name, want.number, want.kind, want.messageType)
				if f.IsList() != (want.name == "versions") || f.ContainingOneof() != nil {
					t.Errorf("unexpected cardinality or oneof for %s", f.FullName())
				}
			}
		})
	}
}

func TestV0230LatePaymentWindowSourceAndLimit(t *testing.T) {
	source := requireField(t, &reconciliationv1.GetLatePaymentWindowResponse{}, "source", 1, protoreflect.EnumKind, "").Enum()
	if source.FullName() != "aipay.reconciliation.v1.LatePaymentWindowSource" {
		t.Fatalf("unexpected source enum %s", source.FullName())
	}
	for number, name := range []protoreflect.Name{
		"LATE_PAYMENT_WINDOW_SOURCE_UNSPECIFIED",
		"LATE_PAYMENT_WINDOW_SOURCE_ENVIRONMENT",
		"LATE_PAYMENT_WINDOW_SOURCE_VERSION",
	} {
		value := source.Values().ByNumber(protoreflect.EnumNumber(number))
		if value == nil || value.Name() != name {
			t.Errorf("window source %d must be %s", number, name)
		}
	}
	limit := requireField(t, &reconciliationv1.ListLatePaymentWindowsRequest{}, "limit", 1, protoreflect.Uint32Kind, "")
	if got := proto.GetExtension(limit.Options(), commonv1.E_MaxItems); got != uint32(100) {
		t.Errorf("%s max_items = %v, want 100", limit.FullName(), got)
	}
}

func TestV0230Policies(t *testing.T) {
	admin := reconciliationv1.File_aipay_reconciliation_v1_reconciliation_admin_service_proto.Services().ByName("ReconciliationAdminService")
	control := ledgerv1.File_aipay_ledger_v1_control_service_proto.Services().ByName("LedgerControlService")
	const (
		recent   = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT
		required = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED
	)
	for _, tc := range []struct {
		service    protoreflect.ServiceDescriptor
		method     protoreflect.Name
		permission string
		assertion  authzv1.AssertionRequirement
		maxAge     uint32
	}{
		{admin, "SubmitLatePaymentWindow", "reconciliation.late_window.submit", recent, 600},
		{admin, "ApproveLatePaymentWindow", "reconciliation.late_window.approve", recent, 600},
		{admin, "GetLatePaymentWindow", "reconciliation.late_window.read", required, 0},
		{admin, "ListLatePaymentWindows", "reconciliation.late_window.read", required, 0},
		{control, "RequestUsdRatePolicy", "ledger.rate_policy.submit", required, 0},
		{control, "ApproveUsdRatePolicy", "ledger.rate_policy.approve", required, 0},
		{control, "GetUsdRatePolicy", "ledger.rate_policy.read", required, 0},
	} {
		t.Run(string(tc.method), func(t *testing.T) {
			method := tc.service.Methods().ByName(tc.method)
			if method == nil || method.IsStreamingClient() || method.IsStreamingServer() || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
				t.Fatal("expected unary RPC with explicit policy")
			}
			pkg := tc.service.ParentFile().Package()
			if method.Input().FullName() != pkg.Append(tc.method+"Request") || method.Output().FullName() != pkg.Append(tc.method+"Response") {
				t.Fatal("unexpected RPC input or output")
			}
			want := &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{{
				Callers: []string{"backend"}, Assertion: tc.assertion,
				Permissions: []string{tc.permission}, RecentAuthMaxAgeSeconds: tc.maxAge,
			}}}
			got := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
			if !proto.Equal(got, want) {
				t.Fatalf("policy = %v, want %v", got, want)
			}
		})
	}
}

func TestV0230CancellationWindowVersionWire(t *testing.T) {
	event := &reconciliationv1.TradeCancellationAuthorized{
		AuthorizationId: "a", Trade: &commonv1.TradeSnapshot{OrderId: "o"},
		EscrowReservationId: "e", ControlVersion: 7,
	}
	field := requireField(t, event, "late_payment_window_version", 9, protoreflect.Uint64Kind, "")
	if field.HasPresence() || field.IsList() {
		t.Fatal("late_payment_window_version must be a singular implicit-presence field")
	}
	marshal := proto.MarshalOptions{Deterministic: true}
	before, err := marshal.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	legacy := []byte{0x0a, 1, 'a', 0x12, 3, 0x0a, 1, 'o', 0x1a, 1, 'e', 0x20, 7}
	if !bytes.Equal(before, legacy) {
		t.Fatalf("original cancellation payload = %x, want %x", before, legacy)
	}
	event.LatePaymentWindowVersion = 0
	zero, err := marshal.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(zero, before) {
		t.Fatalf("zero window version changed the payload from %x to %x", before, zero)
	}
	event.LatePaymentWindowVersion = 1
	versioned, err := marshal.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if want := append(legacy, 0x48, 1); !bytes.Equal(versioned, want) {
		t.Fatalf("versioned cancellation payload = %x, want %x", versioned, want)
	}
}
