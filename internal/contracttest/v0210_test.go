package contracttest

import (
	"testing"

	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// A144: the cancellation trigger states its basis. Zero keeps the original
// meaning and is not encoded, so earlier payloads keep their bytes.
func TestV0210CancellationBasis(t *testing.T) {
	msg := &paymentv1.OrderCancellationVerificationRequested{}
	basis := requireField(t, msg, "basis", 4, protoreflect.EnumKind, "")
	if basis.IsList() || basis.ContainingOneof() != nil || basis.HasPresence() {
		t.Fatalf("%s must be a single implicit-presence enum", basis.FullName())
	}
	enum := basis.Enum()
	if enum.FullName() != "aipay.payment.v1.OrderCancellationBasis" {
		t.Fatalf("unexpected cancellation basis %s", enum.FullName())
	}
	for number, name := range []protoreflect.Name{
		"ORDER_CANCELLATION_BASIS_UNSPECIFIED",
		"ORDER_CANCELLATION_BASIS_CREATE_NOT_FOUND_INFERRED",
	} {
		value := enum.Values().ByNumber(protoreflect.EnumNumber(number))
		if value == nil || value.Name() != name {
			t.Errorf("cancellation basis %d must be %s", number, name)
		}
	}
	plain := &paymentv1.OrderCancellationVerificationRequested{OrderId: "o", EscrowReservationId: "e", ControlVersion: 7}
	before, err := proto.MarshalOptions{Deterministic: true}.Marshal(plain)
	if err != nil {
		t.Fatal(err)
	}
	legacy := []byte{0x0a, 0x01, 'o', 0x12, 0x01, 'e', 0x18, 0x07}
	if string(before) != string(legacy) {
		t.Fatalf("an unspecified basis changed the v1 payload bytes: %x", before)
	}
	inferred := proto.Clone(plain).(*paymentv1.OrderCancellationVerificationRequested)
	inferred.Basis = paymentv1.OrderCancellationBasis_ORDER_CANCELLATION_BASIS_CREATE_NOT_FOUND_INFERRED
	after, err := proto.MarshalOptions{Deterministic: true}.Marshal(inferred)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(append(legacy, 0x20, 0x01)) {
		t.Fatalf("unexpected inferred payload bytes: %x", after)
	}
}
