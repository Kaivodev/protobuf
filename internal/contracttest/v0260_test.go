package contracttest

import (
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0260ReportDisputeRulingReview(t *testing.T) {
	service := paymentv1.File_aipay_payment_v1_dispute_service_proto.Services().ByName("DisputeService")
	if service == nil {
		t.Fatal("missing DisputeService")
	}
	method := service.Methods().ByName("ReportDisputeRulingReview")
	if method == nil || method.IsStreamingClient() || method.IsStreamingServer() ||
		method.Input().FullName() != "aipay.payment.v1.ReportDisputeRulingReviewRequest" ||
		method.Output().FullName() != "aipay.payment.v1.ReportDisputeRulingReviewResponse" {
		t.Fatal("unexpected ReportDisputeRulingReview signature")
	}
	want := &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{{
		Callers:   []string{"reconciliation"},
		Assertion: authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED,
	}}}
	got := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
	if !proto.Equal(got, want) {
		t.Errorf("%s policy = %v, want %v", method.FullName(), got, want)
	}
}

func TestV0260DisputePaymentFields(t *testing.T) {
	const (
		msg         = protoreflect.MessageKind
		enum        = protoreflect.EnumKind
		str         = protoreflect.StringKind
		timestamp   = "google.protobuf.Timestamp"
		payment     = "aipay.payment.v1."
		disposition = payment + "DisputeRulingReviewDisposition"
		state       = payment + "DisputePaymentState"
	)
	type fields []struct {
		number   protoreflect.FieldNumber
		name     protoreflect.Name
		kind     protoreflect.Kind
		typeName protoreflect.FullName
		optional bool
	}
	for _, tc := range []struct {
		message    proto.Message
		fieldCount int
		fields     fields
	}{
		{&paymentv1.ReportDisputeRulingReviewRequest{}, 8, fields{
			{1, "ruling_id", str, "", false},
			{2, "dispute_id", str, "", false},
			{3, "order_id", str, "", false},
			{4, "escrow_reservation_id", str, "", false},
			{5, "control_version", protoreflect.Uint64Kind, "", false},
			{6, "disposition", enum, disposition, false},
			{7, "reason", str, "", false},
			{8, "reviewed_at", msg, timestamp, false},
		}},
		{&paymentv1.ReportDisputeRulingReviewResponse{}, 1, fields{
			{1, "recorded", protoreflect.BoolKind, "", false},
		}},
		{&paymentv1.DisputePaymentObservation{}, 7, fields{
			{1, "state", enum, state, false},
			{2, "upstream_order_id", str, "", false},
			{3, "query_evidence_sha256", protoreflect.BytesKind, "", false},
			{4, "observed_at", msg, timestamp, false},
			{5, "resolved_at", msg, timestamp, false},
			{6, "resolution", enum, payment + "DisputePaymentResolution", false},
			{7, "late_payment_id", str, "", false},
		}},
		{&paymentv1.DisputeRulingReview{}, 4, fields{
			{1, "disposition", enum, disposition, false},
			{2, "reason", str, "", false},
			{3, "reviewed_at", msg, timestamp, false},
			{4, "recorded_at", msg, timestamp, false},
		}},
		{&paymentv1.PlatformDisputeView{}, 9, fields{
			{8, "payment_observation", msg, payment + "DisputePaymentObservation", false},
			{9, "ruling_review", msg, payment + "DisputeRulingReview", false},
		}},
		{&paymentv1.SellerDisputeView{}, 15, fields{
			{15, "payment_state", enum, state, true},
		}},
		{&paymentv1.LatePayment{}, 13, fields{
			{13, "dispute_id", str, "", true},
		}},
	} {
		desc := tc.message.ProtoReflect().Descriptor()
		t.Run(string(desc.Name()), func(t *testing.T) {
			if got := desc.Fields().Len(); got != tc.fieldCount {
				t.Fatalf("field count = %d, want %d", got, tc.fieldCount)
			}
			for _, want := range tc.fields {
				field := requireField(t, tc.message, want.name, want.number, want.kind, "")
				if field.Cardinality() != protoreflect.Optional || field.HasOptionalKeyword() != want.optional ||
					field.HasPresence() != (want.optional || want.kind == msg) {
					t.Errorf("unexpected cardinality or presence for %s", field.FullName())
				}
				if group := field.ContainingOneof(); group != nil && !group.IsSynthetic() {
					t.Errorf("%s unexpectedly belongs to %s", field.FullName(), group.FullName())
				}
				if want.kind == msg && field.Message().FullName() != want.typeName ||
					want.kind == enum && field.Enum().FullName() != want.typeName {
					t.Errorf("%s must use %s", field.FullName(), want.typeName)
				}
			}
		})
	}
}

func TestV0260DisputePaymentEnums(t *testing.T) {
	for _, tc := range []struct {
		enum  protoreflect.EnumDescriptor
		names []protoreflect.Name // Indexed by wire number.
	}{
		{paymentv1.DisputeRulingReviewDisposition(0).Descriptor(), []protoreflect.Name{
			"DISPUTE_RULING_REVIEW_DISPOSITION_UNSPECIFIED",
			"DISPUTE_RULING_REVIEW_DISPOSITION_DENIED",
			"DISPUTE_RULING_REVIEW_DISPOSITION_QUARANTINED",
		}},
		{paymentv1.DisputePaymentState(0).Descriptor(), []protoreflect.Name{
			"DISPUTE_PAYMENT_STATE_UNSPECIFIED",
			"DISPUTE_PAYMENT_STATE_FENCED",
			"DISPUTE_PAYMENT_STATE_VERIFIED",
			"DISPUTE_PAYMENT_STATE_LATE_PAYMENT",
		}},
		{paymentv1.DisputePaymentResolution(0).Descriptor(), []protoreflect.Name{
			"DISPUTE_PAYMENT_RESOLUTION_UNSPECIFIED",
			"DISPUTE_PAYMENT_RESOLUTION_NO_RULING",
			"DISPUTE_PAYMENT_RESOLUTION_RECONCILIATION_DENIED",
			"DISPUTE_PAYMENT_RESOLUTION_RULING_EXPIRED",
			"DISPUTE_PAYMENT_RESOLUTION_LEDGER_CANCELLED",
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
