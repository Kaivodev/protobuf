package contracttest

import (
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	reconciliationv1 "github.com/Kaivodev/protobuf/gen/go/aipay/reconciliation/v1"
	upstreamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/upstream/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0180StoreChannelCredentials(t *testing.T) {
	service := paymentv1.File_aipay_payment_v1_channel_admin_service_proto.Services().ByName("ChannelAdminService")
	method := service.Methods().ByName("StoreChannelCredentials")
	if method == nil || method.IsStreamingClient() || method.IsStreamingServer() ||
		method.Input().FullName() != "aipay.payment.v1.StoreChannelCredentialsRequest" ||
		method.Output().FullName() != "aipay.payment.v1.StoreChannelCredentialsResponse" {
		t.Fatal("unexpected StoreChannelCredentials signature")
	}
	if method.Input().Fields().Len() != 4 || method.Output().Fields().Len() != 2 {
		t.Fatal("unexpected StoreChannelCredentials request or response fields")
	}
	if !proto.HasExtension(method.Options(), authzv1.E_Policy) {
		t.Fatal("missing StoreChannelCredentials policy")
	}
	policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
	want := &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{{
		Callers:     []string{"backend"},
		Assertion:   authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED,
		Permissions: []string{"payment.channel.manage"},
	}}}
	if !proto.Equal(policy, want) {
		t.Fatalf("unexpected StoreChannelCredentials policy: %v", policy)
	}
}

func TestV0180Fields(t *testing.T) {
	request := &paymentv1.StoreChannelCredentialsRequest{}
	response := &paymentv1.StoreChannelCredentialsResponse{}
	for _, tc := range []struct {
		msg      proto.Message
		name     protoreflect.Name
		number   protoreflect.FieldNumber
		kind     protoreflect.Kind
		optional bool
	}{
		{request, "idempotency_key", 1, protoreflect.StringKind, false},
		{request, "channel_account_id", 2, protoreflect.StringKind, false},
		{request, "channel_code", 3, protoreflect.StringKind, false},
		{request, "credentials_json", 4, protoreflect.BytesKind, false},
		{response, "credentials_ref", 1, protoreflect.StringKind, false},
		{response, "credentials_sha256", 2, protoreflect.BytesKind, false},
		{&reconciliationv1.StoreVerificationCredentialsResponse{}, "credentials_sha256", 2, protoreflect.BytesKind, false},
		{&paymentv1.ChannelAccountConfiguration{}, "credentials_sha256", 17, protoreflect.BytesKind, true},
		{&reconciliationv1.VerificationAccountConfiguration{}, "credentials_sha256", 6, protoreflect.BytesKind, true},
		{&upstreamv1.CreatePaymentInput{}, "client_ip", 7, protoreflect.StringKind, true},
		{&upstreamv1.CreatePaymentInput{}, "device", 8, protoreflect.StringKind, true},
		{&paymentv1.CreateOrderRequest{}, "client_ip", 11, protoreflect.StringKind, true},
		{&paymentv1.CreateOrderRequest{}, "device", 12, protoreflect.StringKind, true},
	} {
		field := requireField(t, tc.msg, tc.name, tc.number, tc.kind, "")
		if field.IsList() || field.HasOptionalKeyword() != tc.optional || field.HasPresence() != tc.optional {
			t.Errorf("%s must be singular with explicit presence %t", field.FullName(), tc.optional)
		}
	}
}
