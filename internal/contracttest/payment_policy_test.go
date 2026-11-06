package contracttest

import (
	"slices"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// A syntactically valid policy must not silently grant backend service-only
// order/refund access, notifier payment authority, or extra management RPCs.
func TestPaymentMethodAuthorization(t *testing.T) {
	services := []struct {
		file       protoreflect.FileDescriptor
		name       protoreflect.Name
		caller     string
		assertion  authzv1.AssertionRequirement
		permission map[protoreflect.Name]string
	}{
		{
			paymentv1.File_aipay_payment_v1_order_service_proto, "OrderService", "backend",
			authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED,
			map[protoreflect.Name]string{
				"CreateOrder": "payment.order.create", "GetOrder": "payment.order.read",
				"CloseOrder": "payment.order.close", "ListOrders": "payment.order.read",
			},
		},
		{
			paymentv1.File_aipay_payment_v1_refund_service_proto, "RefundService", "backend",
			authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED,
			map[protoreflect.Name]string{
				"CreateRefund": "payment.refund.create", "GetRefund": "payment.refund.read",
			},
		},
		{
			paymentv1.File_aipay_payment_v1_callback_service_proto, "CallbackService", "backend",
			authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED,
			map[protoreflect.Name]string{"HandleUpstreamCallback": ""},
		},
		{
			paymentv1.File_aipay_payment_v1_merchant_payment_access_service_proto, "MerchantPaymentAccessService", "backend",
			authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED,
			map[protoreflect.Name]string{
				"List": "payment.access.read", "Get": "payment.access.read",
				"Submit": "payment.access.manage", "Review": "payment.access.review",
				"SetEnabled": "payment.access.manage", "Suspend": "payment.access.suspend", "Revoke": "payment.access.suspend",
			},
		},
		{
			paymentv1.File_aipay_payment_v1_notification_delivery_service_proto, "NotificationDeliveryService", "notifier",
			authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED,
			map[protoreflect.Name]string{"LeaseNotifications": "", "ReportDelivery": ""},
		},
	}
	for _, want := range services {
		t.Run(string(want.name), func(t *testing.T) {
			if want.file.Services().Len() != 1 {
				t.Fatal("unreviewed service set")
			}
			service := want.file.Services().ByName(want.name)
			if service == nil || service.Methods().Len() != len(want.permission) {
				t.Fatal("unreviewed RPC set")
			}
			for name, permission := range want.permission {
				t.Run(string(name), func(t *testing.T) {
					method := service.Methods().ByName(name)
					if method == nil || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
						t.Fatal("missing method or policy")
					}
					policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
					if len(policy.GetRules()) != 1 {
						t.Fatalf("unexpected caller rules: %v", policy)
					}
					rule := policy.GetRules()[0]
					var permissions []string
					if permission != "" {
						permissions = []string{permission}
					}
					if !slices.Equal(rule.GetCallers(), []string{want.caller}) || rule.GetAssertion() != want.assertion || !slices.Equal(rule.GetPermissions(), permissions) || rule.GetRecentAuthMaxAgeSeconds() != 0 {
						t.Fatalf("unexpected authorization: %v", rule)
					}
				})
			}
		})
	}
}

// Explicit zero risk/fee and a merchant disabling access must survive transport
// without becoming indistinguishable from missing required configuration.
func TestPaymentExplicitZeroPresence(t *testing.T) {
	for _, message := range []proto.Message{
		&paymentv1.RiskPolicyApproved{HoldDurationSeconds: proto.Uint64(0)},
		&paymentv1.PricingConfiguration{MarkupPpm: proto.Int32(0), FeeRatePpm: proto.Int32(0)},
		&paymentv1.SetEnabledRequest{Enabled: proto.Bool(false)},
		&paymentv1.RiskPolicyChange{HoldDurationSeconds: proto.Uint64(0)},
		&paymentv1.ChannelAccountConfiguration{Enabled: proto.Bool(false)},
		&paymentv1.PaymentMethodChange{Enabled: proto.Bool(false)},
	} {
		t.Run(string(message.ProtoReflect().Descriptor().Name()), func(t *testing.T) {
			encoded, err := proto.Marshal(message)
			if err != nil {
				t.Fatal(err)
			}
			decoded := message.ProtoReflect().Type().New().Interface()
			if err := proto.Unmarshal(encoded, decoded); err != nil {
				t.Fatal(err)
			}
			message.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
				if !field.HasPresence() || !decoded.ProtoReflect().Has(field) || decoded.ProtoReflect().Get(field).Interface() != value.Interface() {
					t.Errorf("lost explicit zero for %s", field.FullName())
				}
				if message.ProtoReflect().Type().New().Has(field) {
					t.Errorf("absent field %s has presence", field.FullName())
				}
				return true
			})
		})
	}
}
