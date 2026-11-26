package contracttest

import (
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0190RefundPolicies(t *testing.T) {
	review := paymentv1.File_aipay_payment_v1_seller_refund_proto.Services().ByName("SellerRefundReviewService")
	seller := paymentv1.File_aipay_payment_v1_seller_portal_service_proto.Services().ByName("SellerPortalService")
	admin := paymentv1.File_aipay_payment_v1_payment_admin_service_proto.Services().ByName("PaymentAdminService")
	if review == nil || review.Methods().Len() != 8 {
		t.Fatal("expected eight seller refund review methods")
	}

	const (
		required = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED
		recent   = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT
		workload = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED
	)
	for _, tc := range []struct {
		service    protoreflect.ServiceDescriptor
		method     protoreflect.Name
		caller     string
		assertion  authzv1.AssertionRequirement
		permission string
	}{
		{seller, "ListSellerLatePayments", "backend", required, "payment.refund.read"},
		{seller, "GetSellerLatePayment", "backend", required, "payment.refund.read"},
		{seller, "SubmitSellerRefundEvidence", "backend", recent, "payment.refund.process"},
		{seller, "GetSellerRefundEvidenceFile", "backend", required, "payment.refund.read"},
		{admin, "CreatePlatformRefund", "backend", recent, "payment.refund.initiate"},
		{admin, "ListLatePayments", "backend", required, "payment.refund.read"},
		{admin, "GetLatePayment", "backend", required, "payment.refund.read"},
		{review, "GetSellerRefundReview", "backend", required, "payment.refund.review.read"},
		{review, "GetSellerRefundEvidenceFile", "backend", required, "payment.refund.review.read"},
		{review, "ApproveSellerRefund", "backend", required, "payment.refund.review"},
		{review, "RejectSellerRefundEvidence", "backend", required, "payment.refund.review"},
		{review, "RecordRefundObjection", "backend", required, "payment.refund.review"},
		{review, "SubmitSellerRefundFailure", "backend", required, "payment.refund.fail.submit"},
		{review, "ApproveSellerRefundFailure", "backend", required, "payment.refund.review"},
		{review, "GetSellerRefundReviewForVerification", "reconciliation", workload, ""},
	} {
		t.Run(string(tc.service.Name())+"/"+string(tc.method), func(t *testing.T) {
			method := tc.service.Methods().ByName(tc.method)
			if method == nil || method.IsStreamingClient() || method.IsStreamingServer() || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
				t.Fatal("expected a unary RPC with an explicit policy")
			}
			rule := &authzv1.CallerRule{Callers: []string{tc.caller}, Assertion: tc.assertion}
			if tc.permission != "" {
				rule.Permissions = []string{tc.permission}
			}
			if tc.assertion == recent {
				rule.RecentAuthMaxAgeSeconds = 600
			}
			got := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
			if !proto.Equal(got, &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule}}) {
				t.Fatalf("unexpected caller rules: %v", got)
			}
		})
	}
}
