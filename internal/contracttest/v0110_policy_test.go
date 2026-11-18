package contracttest

import (
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	iamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/iam/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0110MethodAuthorization(t *testing.T) {
	serviceOnly := func(caller string) *authzv1.CallerRule {
		return &authzv1.CallerRule{Callers: []string{caller}, Assertion: authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED}
	}
	replaceTotp := &authzv1.CallerRule{
		Callers: []string{"backend"}, Assertion: authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT,
		Permissions: []string{"iam.mfa.manage"}, RecentAuthMaxAgeSeconds: 600,
	}
	for _, tc := range []struct {
		file            protoreflect.FileDescriptor
		service, method protoreflect.Name
		rule            *authzv1.CallerRule
	}{
		{ledgerv1.File_aipay_ledger_v1_reservation_service_proto, "ReservationService", "GetTradeFreezeDetails", serviceOnly("reconciliation")},
		{paymentv1.File_aipay_payment_v1_callback_service_proto, "CallbackService", "GetCallbackPolicy", serviceOnly("backend")},
		{iamv1.File_aipay_iam_v1_authorization_proto, "AuthorizationService", "Authorize", serviceOnly("backend")},
		{iamv1.File_aipay_iam_v1_account_service_proto, "AccountService", "Login", serviceOnly("backend")},
		{iamv1.File_aipay_iam_v1_account_service_proto, "AccountService", "GetChallenge", serviceOnly("backend")},
		{iamv1.File_aipay_iam_v1_account_service_proto, "AccountService", "VerifyChallenge", serviceOnly("backend")},
		{iamv1.File_aipay_iam_v1_mfa_service_proto, "MfaService", "BeginTotpReplacement", replaceTotp},
		{iamv1.File_aipay_iam_v1_mfa_service_proto, "MfaService", "ConfirmTotpReplacement", replaceTotp},
	} {
		t.Run(string(tc.method), func(t *testing.T) {
			service := tc.file.Services().ByName(tc.service)
			if service == nil {
				t.Fatalf("missing %s", tc.service)
			}
			method := service.Methods().ByName(tc.method)
			if method == nil || method.IsStreamingClient() || method.IsStreamingServer() || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
				t.Fatal("expected unary method with an explicit policy")
			}
			got := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
			want := &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{tc.rule}}
			if !proto.Equal(got, want) {
				t.Fatalf("policy = %v, want %v", got, want)
			}
		})
	}
}
