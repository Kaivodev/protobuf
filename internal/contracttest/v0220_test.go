package contracttest

import (
	"bytes"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0220ForcedCloseEnums(t *testing.T) {
	for _, tc := range []struct {
		enum  protoreflect.EnumDescriptor
		names []protoreflect.Name // Indexed by wire number.
	}{
		{paymentv1.OrderCancellationBasis(0).Descriptor(), []protoreflect.Name{
			"ORDER_CANCELLATION_BASIS_UNSPECIFIED",
			"ORDER_CANCELLATION_BASIS_CREATE_NOT_FOUND_INFERRED",
			"ORDER_CANCELLATION_BASIS_OPERATOR_FORCED_CLOSE",
			"ORDER_CANCELLATION_BASIS_OPERATOR_FORCED_CLOSE_BOUND",
		}},
		{paymentv1.ForcedCloseBlocker(0).Descriptor(), []protoreflect.Name{
			"FORCED_CLOSE_BLOCKER_UNSPECIFIED",
			"FORCED_CLOSE_BLOCKER_ORDER_STATUS",
			"FORCED_CLOSE_BLOCKER_ORDER_STAGE",
			"FORCED_CLOSE_BLOCKER_CLOSE_IN_PROGRESS",
			"FORCED_CLOSE_BLOCKER_DISPATCH_NOT_GRANTED",
			"FORCED_CLOSE_BLOCKER_ESCROW_NOT_RESERVED",
			"FORCED_CLOSE_BLOCKER_LEDGER_VERSION_CHANGED",
			"FORCED_CLOSE_BLOCKER_LEDGER_PAID",
			"FORCED_CLOSE_BLOCKER_LEDGER_DISPUTED",
			"FORCED_CLOSE_BLOCKER_PAYMENT_REPORTED",
			"FORCED_CLOSE_BLOCKER_CANCELLATION_REQUESTED",
			"FORCED_CLOSE_BLOCKER_TOO_EARLY",
			"FORCED_CLOSE_BLOCKER_MISSING_ANCHOR",
			"FORCED_CLOSE_BLOCKER_STATE_CHANGED",
		}},
		{paymentv1.ForcedCloseState(0).Descriptor(), []protoreflect.Name{
			"FORCED_CLOSE_STATE_UNSPECIFIED",
			"FORCED_CLOSE_STATE_NONE",
			"FORCED_CLOSE_STATE_PENDING_APPROVAL",
			"FORCED_CLOSE_STATE_VERIFYING",
			"FORCED_CLOSE_STATE_CANCELLED",
			"FORCED_CLOSE_STATE_PAID",
			"FORCED_CLOSE_STATE_FENCED",
		}},
	} {
		values := tc.enum.Values()
		if values.Len() != len(tc.names) {
			t.Errorf("%s has %d values, want %d", tc.enum.Name(), values.Len(), len(tc.names))
		}
		for number, name := range tc.names {
			value := values.ByNumber(protoreflect.EnumNumber(number))
			if value == nil || value.Name() != name {
				t.Errorf("%s value %d must be %s", tc.enum.Name(), number, name)
			}
		}
	}
}

func TestV0220ForcedClosePayload(t *testing.T) {
	legacy := []byte{0x0a, 0x01, 'o', 0x12, 0x01, 'e', 0x18, 0x07}
	for _, tc := range []struct {
		basis paymentv1.OrderCancellationBasis
		value byte
	}{
		{paymentv1.OrderCancellationBasis_ORDER_CANCELLATION_BASIS_OPERATOR_FORCED_CLOSE, 0x02},
		{paymentv1.OrderCancellationBasis_ORDER_CANCELLATION_BASIS_OPERATOR_FORCED_CLOSE_BOUND, 0x03},
	} {
		msg := &paymentv1.OrderCancellationVerificationRequested{
			OrderId: "o", EscrowReservationId: "e", ControlVersion: 7, Basis: tc.basis,
		}
		got, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}
		if want := append(legacy, 0x20, tc.value); !bytes.Equal(got, want) {
			t.Errorf("%s payload = %x, want %x", tc.basis, got, want)
		}
	}
}

func TestV0220ForcedCloseFields(t *testing.T) {
	operation := requireField(t, &paymentv1.PaymentApprovalMaterial{}, "forced_close", 16, protoreflect.MessageKind, "aipay.payment.v1.OrderForcedCloseMaterial")
	if oneof := operation.ContainingOneof(); oneof == nil || oneof.Name() != "operation" || oneof.IsSynthetic() {
		t.Fatal("forced_close must select the approval operation")
	}
	link := requireField(t, &paymentv1.LatePayment{}, "forced_close_approval_request_id", 12, protoreflect.StringKind, "")
	if !link.HasOptionalKeyword() {
		t.Fatal("late payments must preserve forced close approval ID presence")
	}

	type field struct {
		name    protoreflect.Name
		kind    protoreflect.Kind
		message protoreflect.FullName
	}
	for _, tc := range []struct {
		msg    proto.Message
		fields []field // In field-number order, starting at 1.
	}{
		{&paymentv1.OrderForcedCloseMaterial{}, []field{
			{"trade", protoreflect.MessageKind, "aipay.common.v1.TradeSnapshot"},
			{"escrow_reservation_id", protoreflect.StringKind, ""},
			{"expected_control_version", protoreflect.Uint64Kind, ""},
			{"reason", protoreflect.StringKind, ""},
			{"status", protoreflect.EnumKind, ""},
			{"upstream_order_id", protoreflect.StringKind, ""},
			{"never_sent", protoreflect.BoolKind, ""},
			{"eligible_at", protoreflect.MessageKind, "google.protobuf.Timestamp"},
		}},
		{&paymentv1.GetForcedCloseRequest{}, []field{
			{"order_id", protoreflect.StringKind, ""},
		}},
		{&paymentv1.GetForcedCloseResponse{}, []field{
			{"forced_close", protoreflect.MessageKind, "aipay.payment.v1.ForcedCloseView"},
		}},
		{&paymentv1.SubmitForcedCloseRequest{}, []field{
			{"idempotency_key", protoreflect.StringKind, ""},
			{"order_id", protoreflect.StringKind, ""},
			{"expected_control_version", protoreflect.Uint64Kind, ""},
			{"reason", protoreflect.StringKind, ""},
		}},
		{&paymentv1.SubmitForcedCloseResponse{}, []field{
			{"approval", protoreflect.MessageKind, "aipay.payment.v1.PaymentApproval"},
		}},
		{&paymentv1.ApproveForcedCloseRequest{}, []field{
			{"idempotency_key", protoreflect.StringKind, ""},
			{"approval_request_id", protoreflect.StringKind, ""},
			{"assertion", protoreflect.MessageKind, "aipay.approval.v1.WebAuthnAssertion"},
		}},
		{&paymentv1.ApproveForcedCloseResponse{}, []field{
			{"approval", protoreflect.MessageKind, "aipay.payment.v1.PaymentApproval"},
		}},
	} {
		desc := tc.msg.ProtoReflect().Descriptor()
		if got := desc.Fields().Len(); got != len(tc.fields) {
			t.Errorf("%s has %d fields, want %d", desc.Name(), got, len(tc.fields))
		}
		for i, want := range tc.fields {
			f := requireField(t, tc.msg, want.name, protoreflect.FieldNumber(i+1), want.kind, want.message)
			// Singular message fields always have presence in proto3.
			if f.IsList() || f.ContainingOneof() != nil || f.HasOptionalKeyword() || f.HasPresence() != (want.kind == protoreflect.MessageKind) {
				t.Errorf("unexpected cardinality or presence for %s", f.FullName())
			}
			if want.kind == protoreflect.EnumKind && f.Enum().FullName() != "aipay.payment.v1.OrderStatus" {
				t.Errorf("%s must use OrderStatus", f.FullName())
			}
		}
	}
}

func TestV0220ForcedClosePolicies(t *testing.T) {
	service := paymentv1.File_aipay_payment_v1_forced_close_proto.Services().ByName("OrderForcedCloseService")
	if service == nil || service.Methods().Len() != 3 {
		t.Fatal("expected three order forced close methods")
	}
	const (
		required = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED
		recent   = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT
	)
	for _, tc := range []struct {
		method     protoreflect.Name
		assertion  authzv1.AssertionRequirement
		permission string
		maxAge     uint32
	}{
		{"GetForcedClose", required, "payment.order.force_close.read", 0},
		{"SubmitForcedClose", recent, "payment.order.force_close.submit", 600},
		{"ApproveForcedClose", required, "payment.order.force_close.approve", 0},
	} {
		t.Run(string(tc.method), func(t *testing.T) {
			m := service.Methods().ByName(tc.method)
			if m == nil || m.IsStreamingClient() || m.IsStreamingServer() || !proto.HasExtension(m.Options(), authzv1.E_Policy) {
				t.Fatal("expected a unary RPC with an explicit policy")
			}
			name := service.ParentFile().Package().Append(tc.method)
			if m.Input().FullName() != name+"Request" || m.Output().FullName() != name+"Response" {
				t.Fatal("unexpected RPC input or output")
			}
			want := &authzv1.CallerRule{
				Callers: []string{"backend"}, Assertion: tc.assertion,
				Permissions: []string{tc.permission}, RecentAuthMaxAgeSeconds: tc.maxAge,
			}
			got := proto.GetExtension(m.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
			if !proto.Equal(got, &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{want}}) {
				t.Fatalf("unexpected caller rules: %v", got)
			}
		})
	}
}
