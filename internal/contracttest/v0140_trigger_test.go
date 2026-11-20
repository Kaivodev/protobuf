package contracttest

import (
	"testing"

	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0140DisputeRulingApprovedPayload(t *testing.T) {
	msg := &paymentv1.DisputeRulingApproved{}
	desc := msg.ProtoReflect().Descriptor()
	if desc.FullName() != "aipay.payment.v1.DisputeRulingApproved" || desc.Fields().Len() != 5 {
		t.Fatalf("unexpected ruling trigger %s with %d fields", desc.FullName(), desc.Fields().Len())
	}
	for _, field := range []protoreflect.FieldDescriptor{
		requireField(t, msg, "order_id", 1, protoreflect.StringKind, ""),
		requireField(t, msg, "escrow_reservation_id", 2, protoreflect.StringKind, ""),
		requireField(t, msg, "control_version", 3, protoreflect.Uint64Kind, ""),
		requireField(t, msg, "dispute_id", 4, protoreflect.StringKind, ""),
		requireField(t, msg, "ruling_id", 5, protoreflect.StringKind, ""),
	} {
		if field.IsList() || field.ContainingOneof() != nil {
			t.Fatalf("%s must be a singular field outside a oneof", field.FullName())
		}
	}
}
