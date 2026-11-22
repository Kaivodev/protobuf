package contracttest

import (
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	commonv1 "github.com/Kaivodev/protobuf/gen/go/aipay/common/v1"
	iamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/iam/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0150PaymentAndReportPolicies(t *testing.T) {
	backend := func(permission string) *authzv1.CallerRule {
		return &authzv1.CallerRule{
			Callers: []string{"backend"}, Assertion: authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED,
			Permissions: []string{permission},
		}
	}
	recentBackend := func(permission string) *authzv1.CallerRule {
		rule := backend(permission)
		rule.Assertion = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT
		rule.RecentAuthMaxAgeSeconds = 600
		return rule
	}
	iamMail := &authzv1.CallerRule{Callers: []string{"iam"}, Assertion: authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED}
	seller := paymentv1.File_aipay_payment_v1_seller_portal_service_proto.Services().ByName("SellerPortalService")
	admin := paymentv1.File_aipay_payment_v1_payment_admin_service_proto.Services().ByName("PaymentAdminService")
	report := ledgerv1.File_aipay_ledger_v1_report_service_proto.Services().ByName("ReportService")
	if report == nil || report.Methods().Len() != 2 {
		t.Fatal("unexpected report RPC surface")
	}
	for _, tc := range []struct {
		service protoreflect.ServiceDescriptor
		method  protoreflect.Name
		rules   []*authzv1.CallerRule
	}{
		{seller, "SetSellerChannelOnline", []*authzv1.CallerRule{recentBackend("payment.channel.online")}},
		{admin, "ListArchivedWork", []*authzv1.CallerRule{backend("payment.work.read")}},
		{admin, "RequeueArchivedWork", []*authzv1.CallerRule{recentBackend("payment.work.requeue")}},
		{report, "ListMerchantStatements", []*authzv1.CallerRule{backend("ledger.statement.read")}},
		{report, "GetMerchantStatement", []*authzv1.CallerRule{backend("ledger.statement.read"), iamMail}},
	} {
		t.Run(string(tc.method), func(t *testing.T) {
			method := tc.service.Methods().ByName(tc.method)
			if method == nil || method.IsStreamingClient() || method.IsStreamingServer() || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
				t.Fatal("expected a unary RPC with an explicit policy")
			}
			got := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
			if !proto.Equal(got, &authzv1.MethodPolicy{Rules: tc.rules}) {
				t.Fatalf("unexpected caller rules: %v", got)
			}
		})
	}
}

func TestV0150MessageFields(t *testing.T) {
	for _, tc := range []struct {
		msg   proto.Message
		names []protoreflect.Name
	}{
		{&paymentv1.MerchantPaymentReceived{}, []protoreflect.Name{"account_id", "business_id", "application_id", "order_id", "trade_no", "out_trade_no", "fiat_amount", "fiat_currency", "paid_at"}},
		{&paymentv1.MerchantFundsAvailable{}, []protoreflect.Name{"account_id", "business_id", "application_id", "order_id", "trade_no", "out_trade_no", "recipient_usdt", "released_at"}},
		{&paymentv1.PaymentAccessReviewed{}, []protoreflect.Name{"account_id", "business_id", "payment_access_id", "revision_version", "method", "result", "reviewed_at"}},
		{&paymentv1.ArchivedWork{}, []protoreflect.Name{"kind", "object_type", "object_id", "attempts", "last_error_code", "archived_at", "version"}},
		{&paymentv1.RequeueArchivedWorkRequest{}, []protoreflect.Name{"idempotency_key", "kind", "object_id", "expected_version", "reason"}},
		{&iamv1.AccountNotificationRecipient{}, []protoreflect.Name{"chat_id", "kind"}},
		{&ledgerv1.MerchantStatementScope{}, []protoreflect.Name{"kind", "target_id"}},
		{&ledgerv1.GetMerchantStatementRequest{}, []protoreflect.Name{"business_date", "scope", "account_id"}},
		{&ledgerv1.ListMerchantStatementsRequest{}, []protoreflect.Name{"scope", "page_size", "page_token"}},
	} {
		desc := tc.msg.ProtoReflect().Descriptor()
		fields := desc.Fields()
		if fields.Len() != len(tc.names) {
			t.Errorf("%s has %d fields, want %d", desc.Name(), fields.Len(), len(tc.names))
		}
		for i, name := range tc.names {
			number := protoreflect.FieldNumber(i + 1)
			if f := fields.ByNumber(number); f == nil || f.Name() != name {
				t.Errorf("%s field %d must be %s", desc.Name(), number, name)
			}
		}
	}
}

func TestV0150PresenceAndPageBounds(t *testing.T) {
	for _, field := range []protoreflect.FieldDescriptor{
		requireField(t, &paymentv1.SetSellerChannelOnlineRequest{}, "online", 3, protoreflect.BoolKind, ""),
		requireField(t, &paymentv1.SellerChannelView{}, "online", 6, protoreflect.BoolKind, ""),
		requireField(t, &paymentv1.ChannelAccountView{}, "seller_online", 9, protoreflect.BoolKind, ""),
		requireField(t, &paymentv1.ListArchivedWorkRequest{}, "kind", 3, protoreflect.StringKind, ""),
		requireField(t, &ledgerv1.GetMerchantStatementRequest{}, "account_id", 3, protoreflect.StringKind, ""),
		requireField(t, &ledgerv1.MerchantStatementScope{}, "target_id", 2, protoreflect.StringKind, ""),
	} {
		if !field.HasOptionalKeyword() {
			t.Errorf("%s requires explicit presence", field.FullName())
		}
	}
	for _, field := range []protoreflect.FieldDescriptor{
		requireField(t, &paymentv1.ListArchivedWorkRequest{}, "page_size", 1, protoreflect.Uint32Kind, ""),
		requireField(t, &ledgerv1.ListMerchantStatementsRequest{}, "page_size", 2, protoreflect.Uint32Kind, ""),
	} {
		if proto.GetExtension(field.Options(), commonv1.E_MaxItems).(uint32) != 100 {
			t.Errorf("%s requires a page bound of 100", field.FullName())
		}
	}
}

func TestV0150SellerOnlineViews(t *testing.T) {
	seller := &paymentv1.SellerChannelView{}
	if seller.ProtoReflect().Descriptor().Fields().ByName("seller_online") != nil {
		t.Error("SellerChannelView already exposes seller intent through online")
	}
	requireField(t, seller, "seller_online_version", 15, protoreflect.Uint64Kind, "")
	requireField(t, seller, "seller_online_changed_at", 16, protoreflect.MessageKind, "google.protobuf.Timestamp")
	platform := &paymentv1.ChannelAccountView{}
	requireField(t, platform, "seller_online_version", 10, protoreflect.Uint64Kind, "")
	requireField(t, platform, "seller_online_changed_at", 11, protoreflect.MessageKind, "google.protobuf.Timestamp")
}
