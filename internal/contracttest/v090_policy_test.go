package contracttest

import (
	"slices"
	"strings"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	iamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/iam/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	reconciliationv1 "github.com/Kaivodev/protobuf/gen/go/aipay/reconciliation/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type v090PolicyGroup struct {
	file    protoreflect.FileDescriptor
	service protoreflect.Name
	// New service surfaces are exhaustive. Reconciliation also retains 13 v0.8 RPCs.
	retained int
	methods  map[protoreflect.Name]string
}

func v090Groups() []v090PolicyGroup {
	return []v090PolicyGroup{
		{paymentv1.File_aipay_payment_v1_payment_admin_service_proto, "PaymentAdminService", 0, map[protoreflect.Name]string{
			"ListChannelAccounts": "payment.channel.read", "GetChannelAccount": "payment.channel.read",
			"ListPaymentMethods": "payment.method.read", "GetPaymentMethod": "payment.method.read",
			"ListRiskPolicies": "payment.risk.read", "GetRiskPolicy": "payment.risk.read",
			"ListPricingDrafts": "payment.pricing.read", "GetPricingDraft": "payment.pricing.read",
			"ListPricingVersions": "payment.pricing.read", "GetPricingVersion": "payment.pricing.read",
			"ListPlugins": "payment.plugin.read", "GetPlugin": "payment.plugin.read",
			"ListPaymentAccesses": "payment.access.read", "GetPaymentAccess": "payment.access.read",
			"ListPlatformOrders": "payment.order.read", "GetPlatformOrder": "payment.order.read",
			"ListPlatformRefunds": "payment.refund.read", "GetPlatformRefund": "payment.refund.read",
			"ListPlatformDisputes": "payment.dispute.read", "GetPlatformDispute": "payment.dispute.read",
			"ListApprovals": "payment.approval.read", "GetApproval": "payment.approval.read",
			"EmergencyDisablePlugin": "payment.plugin.disable",
		}},
		{paymentv1.File_aipay_payment_v1_seller_portal_service_proto, "SellerPortalService", 0, map[protoreflect.Name]string{
			"ListSellerChannels": "payment.channel.read", "GetSellerChannel": "payment.channel.read",
			"ListSellerOrders": "payment.order.read", "GetSellerOrder": "payment.order.read",
			"ListSellerRefunds": "payment.refund.read", "GetSellerRefund": "payment.refund.read",
			"ListSellerDisputes": "payment.dispute.read", "GetSellerDispute": "payment.dispute.read",
			"AppendSellerEvidence": "payment.dispute.evidence",
		}},
		{ledgerv1.File_aipay_ledger_v1_query_service_proto, "LedgerQueryService", 0, map[protoreflect.Name]string{
			"BatchGetSellerInventory":  "",
			"GetPlatformSellerBalance": "ledger.balance.read", "GetSellerBalance": "ledger.balance.read",
			"ListPlatformSellerDeposits": "ledger.deposit.read", "ListSellerDeposits": "ledger.deposit.read",
			"ListPlatformSellerHolds": "ledger.trade.read", "ListSellerHolds": "ledger.trade.read",
			"GetPlatformSellerHold": "ledger.trade.read", "GetSellerHold": "ledger.trade.read",
		}},
		{reconciliationv1.File_aipay_reconciliation_v1_reconciliation_admin_service_proto, "ReconciliationAdminService", 13, map[protoreflect.Name]string{
			"ListApprovals": "reconciliation.approval.read", "EmergencyDisablePlugin": "reconciliation.plugin.disable",
		}},
		{iamv1.File_aipay_iam_v1_notification_signing_service_proto, "NotificationSigningService", 0, map[protoreflect.Name]string{
			"SignMerchantNotification": "",
		}},
	}
}

// Descriptor contracts only: they do not prove runtime USER/entry/scope checks,
// signature verification, pagination enforcement, atomic fences or idempotency.
func TestV090MethodAuthorization(t *testing.T) {
	for _, group := range v090Groups() {
		t.Run(string(group.service), func(t *testing.T) {
			service := group.file.Services().ByName(group.service)
			if group.file.Services().Len() != 1 || service == nil || service.Methods().Len() != group.retained+len(group.methods) {
				t.Fatal("unreviewed RPC surface")
			}
			for name, permission := range group.methods {
				t.Run(string(name), func(t *testing.T) {
					method := service.Methods().ByName(name)
					if method == nil || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
						t.Fatal("missing RPC or policy")
					}
					policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
					if len(policy.GetRules()) != 1 {
						t.Fatal("unreviewed caller rule")
					}
					caller, assertion := "backend", authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED
					var permissions []string
					var maxAge uint32
					if permission == "" {
						caller, assertion = "payment", authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED
					} else {
						permissions = []string{permission}
					}
					if name == "EmergencyDisablePlugin" {
						assertion, maxAge = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT, 600
					}
					rule := policy.GetRules()[0]
					if !slices.Equal(rule.GetCallers(), []string{caller}) || rule.GetAssertion() != assertion || !slices.Equal(rule.GetPermissions(), permissions) || rule.GetRecentAuthMaxAgeSeconds() != maxAge {
						t.Fatalf("unexpected policy: %v", rule)
					}
					if method.IsStreamingClient() || method.IsStreamingServer() {
						t.Fatal("portal and signing methods must remain unary")
					}
				})
			}
		})
	}
}

func v090SellerMethod(service protoreflect.Name, method protoreflect.Name) bool {
	return service == "SellerPortalService" || service == "LedgerQueryService" && (strings.HasPrefix(string(method), "ListSeller") || strings.HasPrefix(string(method), "GetSeller"))
}
