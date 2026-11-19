package contracttest

import (
	"testing"

	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0120CancellationTriggerPayload(t *testing.T) {
	msg := &paymentv1.OrderCancellationVerificationRequested{}
	desc := msg.ProtoReflect().Descriptor()
	if desc.FullName() != "aipay.payment.v1.OrderCancellationVerificationRequested" || desc.Fields().Len() != 3 {
		t.Fatalf("unexpected cancellation trigger %s with %d fields", desc.FullName(), desc.Fields().Len())
	}
	for _, field := range []protoreflect.FieldDescriptor{
		requireField(t, msg, "order_id", 1, protoreflect.StringKind, ""),
		requireField(t, msg, "escrow_reservation_id", 2, protoreflect.StringKind, ""),
		requireField(t, msg, "control_version", 3, protoreflect.Uint64Kind, ""),
	} {
		if field.IsList() || field.ContainingOneof() != nil {
			t.Fatalf("%s must identify one order, escrow and version together", field.FullName())
		}
	}
}

// Both v2 terminal facts must identify the same refund review and its funding
// source, including when a failure calls for restoring the original reserve.
func TestV0120RefundTriggerBindings(t *testing.T) {
	for _, msg := range []proto.Message{&paymentv1.RefundSucceeded{}, &paymentv1.RefundFailed{}} {
		t.Run(string(msg.ProtoReflect().Descriptor().Name()), func(t *testing.T) {
			requireField(t, msg, "trade", 1, protoreflect.MessageKind, "aipay.common.v1.TradeSnapshot")
			requireField(t, msg, "escrow_reservation_id", 2, protoreflect.StringKind, "")
			requireField(t, msg, "control_version", 3, protoreflect.Uint64Kind, "")
			requireField(t, msg, "refund_id", 4, protoreflect.StringKind, "")
			requireField(t, msg, "refund_reservation_id", 6, protoreflect.StringKind, "")
			branch := requireField(t, msg, "branch", 7, protoreflect.EnumKind, "").Enum()
			requireField(t, msg, "upstream_refund_id", 8, protoreflect.StringKind, "")
			if branch.FullName() != "aipay.ledger.v1.RefundReservationBranch" {
				t.Fatalf("unexpected refund funding source %s", branch.FullName())
			}
			for number, name := range []protoreflect.Name{
				"REFUND_RESERVATION_BRANCH_UNSPECIFIED",
				"REFUND_RESERVATION_BRANCH_PRE_RELEASE",
				"REFUND_RESERVATION_BRANCH_POST_RELEASE",
			} {
				value := branch.Values().ByNumber(protoreflect.EnumNumber(number))
				if value == nil || value.Name() != name {
					t.Errorf("refund branch %d must be %s", number, name)
				}
			}
		})
	}
}
