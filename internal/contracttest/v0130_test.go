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

func TestV0130MerchantBalance(t *testing.T) {
	service := ledgerv1.File_aipay_ledger_v1_balance_service_proto.Services().ByName("BalanceService")
	if service == nil || service.Methods().Len() != 1 {
		t.Fatal("expected only the merchant balance query")
	}
	method := service.Methods().ByName("GetMerchantBalance")
	if method == nil || method.IsStreamingClient() || method.IsStreamingServer() {
		t.Fatal("missing unary GetMerchantBalance")
	}
	if method.Input().FullName() != "aipay.ledger.v1.GetMerchantBalanceRequest" || method.Input().Fields().Len() != 0 {
		t.Fatal("the verified assertion alone selects the account")
	}
	response := &ledgerv1.GetMerchantBalanceResponse{}
	if method.Output().FullName() != response.ProtoReflect().Descriptor().FullName() {
		t.Fatal("expected GetMerchantBalanceResponse")
	}
	balance := &ledgerv1.MerchantBalance{}
	requireField(t, response, "balance", 1, protoreflect.MessageKind, "aipay.ledger.v1.MerchantBalance")
	want := &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{{
		Callers: []string{"backend"}, Assertion: authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED,
		Permissions: []string{"ledger.balance.read", "merchant.profile.read"},
	}}}
	got := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
	if !proto.Equal(got, want) {
		t.Fatalf("GetMerchantBalance policy = %v, want %v", got, want)
	}
	requireField(t, balance, "account_id", 1, protoreflect.StringKind, "")
	requireField(t, balance, "currency", 2, protoreflect.StringKind, "")
	for i, name := range []protoreflect.Name{"available_usdt", "order_escrow_usdt", "risk_held_usdt", "refund_reserved_usdt", "payout_in_flight_usdt", "pending_release_usdt"} {
		requireField(t, balance, name, protoreflect.FieldNumber(i+3), protoreflect.Int64Kind, "")
	}
	requireField(t, balance, "pending_release_orders", 9, protoreflect.Uint64Kind, "")
	requireField(t, balance, "as_of_seq", 10, protoreflect.Uint64Kind, "")
	requireField(t, balance, "as_of", 11, protoreflect.MessageKind, "google.protobuf.Timestamp")
}

func TestV0130ReviewerInvitation(t *testing.T) {
	if role := iamv1.PlatformRole_PLATFORM_ROLE_PAYMENT_ACCESS_REVIEWER; role.Number() != 10 {
		t.Fatalf("reviewer invitation enum = %d, want 10", role)
	}
}

func TestV0130ApplicationNotificationTargets(t *testing.T) {
	targets := requireField(t, &paymentv1.PaymentAccessMaterials{}, "notification_targets", 5, protoreflect.MessageKind, "aipay.payment.v1.ApplicationNotificationTarget")
	if !targets.IsList() {
		t.Fatal("materials must carry per-application targets")
	}
	target := &paymentv1.ApplicationNotificationTarget{}
	requireField(t, target, "application_id", 1, protoreflect.StringKind, "")
	requireField(t, target, "notify_origin", 2, protoreflect.StringKind, "")
	if !requireField(t, target, "return_origin", 3, protoreflect.StringKind, "").HasOptionalKeyword() {
		t.Fatal("the reviewer must be able to distinguish an omitted return origin")
	}
}
