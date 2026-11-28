package contracttest

import (
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	commonv1 "github.com/Kaivodev/protobuf/gen/go/aipay/common/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0280Fields(t *testing.T) {
	type fields []struct {
		name      protoreflect.Name
		fieldType string
	}
	for _, tc := range []struct {
		message proto.Message
		first   protoreflect.FieldNumber
		fields  fields // Consecutive wire numbers starting at first.
	}{
		{&ledgerv1.TradeRiskExtensionPending{}, 1, fields{
			{"extension_id", "string"},
			{"pending_until", "google.protobuf.Timestamp"},
		}},
		{&ledgerv1.TradeReservation{}, 11, fields{
			{"risk_extension_pending", "aipay.ledger.v1.TradeRiskExtensionPending"},
		}},
		{&ledgerv1.HoldTradeForRiskExtensionRequest{}, 1, fields{
			{"idempotency_key", "string"},
			{"order_id", "string"},
			{"escrow_reservation_id", "string"},
			{"expected_control_version", "uint64"},
			{"extension_id", "string"},
			{"pending_until", "google.protobuf.Timestamp"},
			{"reason", "string"},
		}},
		{&ledgerv1.HoldTradeForRiskExtensionResponse{}, 1, fields{
			{"control_version", "uint64"},
		}},
		{&ledgerv1.ResolveTradeRiskExtensionRequest{}, 1, fields{
			{"idempotency_key", "string"},
			{"order_id", "string"},
			{"escrow_reservation_id", "string"},
			{"expected_control_version", "uint64"},
			{"extension_id", "string"},
			{"outcome", "aipay.ledger.v1.TradeRiskExtensionOutcome"},
			{"risk_release_at", "google.protobuf.Timestamp"},
			{"policy_extension_cap_seconds", "optional uint64"},
		}},
		{&ledgerv1.ResolveTradeRiskExtensionResponse{}, 1, fields{
			{"control_version", "uint64"},
		}},
		{&paymentv1.RiskExtensionMaterial{}, 1, fields{
			{"extension_id", "string"},
			{"origin", "aipay.payment.v1.RiskExtensionOrigin"},
			{"trade", "aipay.common.v1.TradeSnapshot"},
			{"escrow_reservation_id", "string"},
			{"expected_control_version", "uint64"},
			{"upstream_order_id", "string"},
			{"paid_verified_at", "google.protobuf.Timestamp"},
			{"current_risk_release_at", "google.protobuf.Timestamp"},
			{"requested_risk_release_at", "google.protobuf.Timestamp"},
			{"extension_seconds", "uint64"},
			{"sequence", "uint32"},
			{"applied_extension_seconds", "uint64"},
			{"risk_policy_version", "string"},
			{"policy_extension_cap_seconds", "uint64"},
			{"pending_until", "google.protobuf.Timestamp"},
			{"reason_code", "aipay.payment.v1.RiskExtensionReason"},
			{"reason", "string"},
		}},
		{&paymentv1.AmountMismatchRecoveryMaterial{}, 1, fields{
			{"recovery_id", "string"},
			{"case_id", "string"},
			{"trade", "aipay.common.v1.TradeSnapshot"},
			{"escrow_reservation_id", "string"},
			{"expected_control_version", "uint64"},
			{"upstream_order_id", "string"},
			{"source", "aipay.payment.v1.AmountMismatchSource"},
			{"kind", "aipay.payment.v1.AmountMismatchKind"},
			{"detected_at", "google.protobuf.Timestamp"},
			{"outcome", "aipay.payment.v1.AmountMismatchOutcome"},
			{"refund_amount", "optional int64"},
			{"refund_id", "optional string"},
			{"reverify_attempt", "optional uint32"},
			{"reason", "string"},
		}},
		{&paymentv1.PaymentApprovalMaterial{}, 17, fields{
			{"risk_extension", "aipay.payment.v1.RiskExtensionMaterial"},
			{"amount_mismatch_recovery", "aipay.payment.v1.AmountMismatchRecoveryMaterial"},
		}},
		{&paymentv1.Refund{}, 18, fields{
			{"amount_mismatch_recovery_id", "string"},
		}},
		{&paymentv1.GetSellerRefundReviewForVerificationResponse{}, 3, fields{
			{"amount_mismatch_recovery", "aipay.payment.v1.PaymentApproval"},
		}},
		{&paymentv1.SellerRefundNotice{}, 11, fields{
			{"origin", "aipay.payment.v1.RefundOrigin"},
		}},
		{&paymentv1.MerchantRefundNotice{}, 13, fields{
			{"origin", "aipay.payment.v1.RefundOrigin"},
		}},
		{&paymentv1.SellerRiskExtensionNotice{}, 1, fields{
			{"seller_account_id", "string"},
			{"order_id", "string"},
			{"trade_no", "string"},
			{"channel_account_id", "string"},
			{"extension_id", "string"},
			{"kind", "aipay.payment.v1.RiskExtensionNoticeKind"},
			{"previous_risk_release_at", "google.protobuf.Timestamp"},
			{"risk_release_at", "google.protobuf.Timestamp"},
			{"occurred_at", "google.protobuf.Timestamp"},
		}},
		{&paymentv1.MerchantRiskExtensionNotice{}, 1, fields{
			{"account_id", "string"},
			{"business_id", "string"},
			{"application_id", "string"},
			{"order_id", "string"},
			{"trade_no", "string"},
			{"out_trade_no", "string"},
			{"risk_release_at", "google.protobuf.Timestamp"},
			{"occurred_at", "google.protobuf.Timestamp"},
		}},
		{&paymentv1.RiskExtension{}, 1, fields{
			{"extension_id", "string"},
			{"order_id", "string"},
			{"origin", "aipay.payment.v1.RiskExtensionOrigin"},
			{"state", "aipay.payment.v1.RiskExtensionState"},
			{"requested_by", "string"},
			{"reason_code", "aipay.payment.v1.RiskExtensionReason"},
			{"extension_seconds", "uint64"},
			{"previous_risk_release_at", "google.protobuf.Timestamp"},
			{"requested_risk_release_at", "google.protobuf.Timestamp"},
			{"pending_until", "google.protobuf.Timestamp"},
			{"approval_request_id", "string"},
			{"created_at", "google.protobuf.Timestamp"},
			{"updated_at", "google.protobuf.Timestamp"},
			{"resolved_at", "google.protobuf.Timestamp"},
		}},
		{&paymentv1.RiskExtensionEligibility{}, 1, fields{
			{"order_id", "string"},
			{"control_version", "uint64"},
			{"eligible", "bool"},
			{"blockers", "repeated aipay.payment.v1.RiskExtensionBlocker"},
			{"paid_verified_at", "google.protobuf.Timestamp"},
			{"current_risk_release_at", "google.protobuf.Timestamp"},
			{"latest_risk_release_at", "google.protobuf.Timestamp"},
			{"remaining_extension_seconds", "uint64"},
			{"applied_count", "uint32"},
			{"max_count", "uint32"},
			{"as_of", "google.protobuf.Timestamp"},
		}},
		{&paymentv1.GetOrderRiskExtensionsRequest{}, 1, fields{
			{"order_id", "string"},
		}},
		{&paymentv1.GetOrderRiskExtensionsResponse{}, 1, fields{
			{"eligibility", "aipay.payment.v1.RiskExtensionEligibility"},
			{"extensions", "repeated aipay.payment.v1.RiskExtension"},
		}},
		{&paymentv1.SubmitRiskExtensionRequest{}, 1, fields{
			{"idempotency_key", "string"},
			{"order_id", "string"},
			{"escrow_reservation_id", "string"},
			{"expected_control_version", "uint64"},
			{"extension_seconds", "uint64"},
			{"reason_code", "aipay.payment.v1.RiskExtensionReason"},
			{"reason", "string"},
		}},
		{&paymentv1.SubmitRiskExtensionResponse{}, 1, fields{
			{"extension", "aipay.payment.v1.RiskExtension"},
			{"approval", "aipay.payment.v1.PaymentApproval"},
		}},
		{&paymentv1.ApproveRiskExtensionRequest{}, 1, fields{
			{"idempotency_key", "string"},
			{"approval_request_id", "string"},
			{"assertion", "aipay.approval.v1.WebAuthnAssertion"},
		}},
		{&paymentv1.ApproveRiskExtensionResponse{}, 1, fields{
			{"approval", "aipay.payment.v1.PaymentApproval"},
			{"extension", "aipay.payment.v1.RiskExtension"},
		}},
		{&paymentv1.RejectRiskExtensionRequest{}, 1, fields{
			{"idempotency_key", "string"},
			{"extension_id", "string"},
			{"reason", "string"},
		}},
		{&paymentv1.RejectRiskExtensionResponse{}, 1, fields{
			{"extension", "aipay.payment.v1.RiskExtension"},
		}},
		{&paymentv1.WithdrawRiskExtensionRequest{}, 1, fields{
			{"idempotency_key", "string"},
			{"extension_id", "string"},
		}},
		{&paymentv1.WithdrawRiskExtensionResponse{}, 1, fields{
			{"extension", "aipay.payment.v1.RiskExtension"},
		}},
		{&paymentv1.ListRiskExtensionsRequest{}, 1, fields{
			{"order_id", "string"},
			{"seller_account_id", "string"},
			{"state", "optional aipay.payment.v1.RiskExtensionState"},
			{"limit", "uint32"},
			{"page_token", "string"},
		}},
		{&paymentv1.ListRiskExtensionsResponse{}, 1, fields{
			{"extensions", "repeated aipay.payment.v1.RiskExtension"},
			{"next_page_token", "string"},
			{"as_of", "google.protobuf.Timestamp"},
		}},
		{&paymentv1.SubmitSellerRiskExtensionRequest{}, 1, fields{
			{"idempotency_key", "string"},
			{"order_id", "string"},
			{"escrow_reservation_id", "string"},
			{"expected_control_version", "uint64"},
			{"extension_seconds", "uint64"},
			{"reason_code", "aipay.payment.v1.RiskExtensionReason"},
			{"reason", "string"},
		}},
		{&paymentv1.SubmitSellerRiskExtensionResponse{}, 1, fields{
			{"extension", "aipay.payment.v1.RiskExtension"},
		}},
		{&paymentv1.ListSellerRiskExtensionsRequest{}, 1, fields{
			{"order_id", "optional string"},
			{"state", "optional aipay.payment.v1.RiskExtensionState"},
			{"limit", "uint32"},
			{"page_token", "string"},
			{"channel_account_id", "string"},
		}},
		{&paymentv1.ListSellerRiskExtensionsResponse{}, 1, fields{
			{"extensions", "repeated aipay.payment.v1.RiskExtension"},
			{"next_page_token", "string"},
			{"as_of", "google.protobuf.Timestamp"},
		}},
		{&paymentv1.WithdrawSellerRiskExtensionRequest{}, 1, fields{
			{"idempotency_key", "string"},
			{"extension_id", "string"},
		}},
		{&paymentv1.WithdrawSellerRiskExtensionResponse{}, 1, fields{
			{"extension", "aipay.payment.v1.RiskExtension"},
		}},
		{&paymentv1.AmountMismatchCase{}, 1, fields{
			{"case_id", "string"},
			{"order_id", "string"},
			{"source", "aipay.payment.v1.AmountMismatchSource"},
			{"kind", "aipay.payment.v1.AmountMismatchKind"},
			{"state", "aipay.payment.v1.AmountMismatchCaseState"},
			{"opened_at", "google.protobuf.Timestamp"},
			{"reverify_attempts", "uint32"},
			{"resolution", "aipay.payment.v1.AmountMismatchResolution"},
			{"resolved_at", "google.protobuf.Timestamp"},
			{"upstream_order_id", "string"},
			{"disputed", "bool"},
		}},
		{&paymentv1.AmountMismatchRecovery{}, 1, fields{
			{"recovery_id", "string"},
			{"case_id", "string"},
			{"order_id", "string"},
			{"outcome", "aipay.payment.v1.AmountMismatchOutcome"},
			{"state", "aipay.payment.v1.AmountMismatchRecoveryState"},
			{"approval_request_id", "string"},
			{"requested_by", "string"},
			{"refund_amount", "optional int64"},
			{"refund_id", "string"},
			{"created_at", "google.protobuf.Timestamp"},
			{"updated_at", "google.protobuf.Timestamp"},
		}},
		{&paymentv1.AmountMismatchEligibility{}, 1, fields{
			{"order_id", "string"},
			{"control_version", "uint64"},
			{"allowed_outcomes", "repeated aipay.payment.v1.AmountMismatchOutcome"},
			{"blockers", "repeated aipay.payment.v1.AmountMismatchBlocker"},
			{"as_of", "google.protobuf.Timestamp"},
		}},
		{&paymentv1.GetAmountMismatchCaseRequest{}, 1, fields{
			{"order_id", "string"},
		}},
		{&paymentv1.GetAmountMismatchCaseResponse{}, 1, fields{
			{"mismatch_case", "aipay.payment.v1.AmountMismatchCase"},
			{"eligibility", "aipay.payment.v1.AmountMismatchEligibility"},
			{"recoveries", "repeated aipay.payment.v1.AmountMismatchRecovery"},
		}},
		{&paymentv1.ListAmountMismatchCasesRequest{}, 1, fields{
			{"state", "optional aipay.payment.v1.AmountMismatchCaseState"},
			{"seller_account_id", "string"},
			{"channel_account_id", "string"},
			{"limit", "uint32"},
			{"page_token", "string"},
		}},
		{&paymentv1.ListAmountMismatchCasesResponse{}, 1, fields{
			{"cases", "repeated aipay.payment.v1.AmountMismatchCase"},
			{"next_page_token", "string"},
			{"as_of", "google.protobuf.Timestamp"},
		}},
		{&paymentv1.SubmitAmountMismatchRecoveryRequest{}, 1, fields{
			{"idempotency_key", "string"},
			{"order_id", "string"},
			{"case_id", "string"},
			{"escrow_reservation_id", "string"},
			{"expected_control_version", "uint64"},
			{"outcome", "aipay.payment.v1.AmountMismatchOutcome"},
			{"refund_amount", "optional int64"},
			{"reason", "string"},
		}},
		{&paymentv1.SubmitAmountMismatchRecoveryResponse{}, 1, fields{
			{"recovery", "aipay.payment.v1.AmountMismatchRecovery"},
			{"approval", "aipay.payment.v1.PaymentApproval"},
		}},
		{&paymentv1.ApproveAmountMismatchRecoveryRequest{}, 1, fields{
			{"idempotency_key", "string"},
			{"approval_request_id", "string"},
			{"assertion", "aipay.approval.v1.WebAuthnAssertion"},
		}},
		{&paymentv1.ApproveAmountMismatchRecoveryResponse{}, 1, fields{
			{"approval", "aipay.payment.v1.PaymentApproval"},
			{"recovery", "aipay.payment.v1.AmountMismatchRecovery"},
		}},
	} {
		message := tc.message.ProtoReflect().Descriptor()
		t.Run(string(message.Name()), func(t *testing.T) {
			if tc.first == 1 && message.Fields().Len() != len(tc.fields) {
				t.Fatalf("got %d fields, want %d", message.Fields().Len(), len(tc.fields))
			}
			for i, want := range tc.fields {
				number := tc.first + protoreflect.FieldNumber(i)
				field := message.Fields().ByNumber(number)
				if field == nil || field.Name() != want.name {
					t.Errorf("field %d must be %s", number, want.name)
					continue
				}
				fieldType := field.Kind().String()
				switch field.Kind() {
				case protoreflect.MessageKind:
					fieldType = string(field.Message().FullName())
				case protoreflect.EnumKind:
					fieldType = string(field.Enum().FullName())
				}
				switch {
				case field.HasOptionalKeyword():
					fieldType = "optional " + fieldType
				case field.IsList():
					fieldType = "repeated " + fieldType
				}
				if fieldType != want.fieldType {
					t.Errorf("%s type = %s, want %s", field.Name(), fieldType, want.fieldType)
				}
				group := field.ContainingOneof()
				if message.Name() == "PaymentApprovalMaterial" {
					if group == nil || group.IsSynthetic() || group.Name() != "operation" {
						t.Errorf("%s must select the approval operation", field.Name())
					}
				} else if group != nil && !group.IsSynthetic() {
					t.Errorf("%s unexpectedly belongs to %s", field.Name(), group.Name())
				}
				if field.HasPresence() != (field.HasOptionalKeyword() || field.Kind() == protoreflect.MessageKind && !field.IsList()) {
					t.Errorf("%s has unexpected presence", field.Name())
				}
				if field.Name() == "limit" && proto.GetExtension(field.Options(), commonv1.E_MaxItems) != uint32(100) {
					t.Errorf("%s must declare a page limit of 100", field.FullName())
				}
			}
		})
	}
}

func TestV0280Enums(t *testing.T) {
	for _, tc := range []struct {
		enum   protoreflect.EnumDescriptor
		prefix string
		values []string // Indexed by wire number.
	}{
		{ledgerv1.TradeRiskExtensionOutcome(0).Descriptor(), "TRADE_RISK_EXTENSION_OUTCOME_", []string{
			"UNSPECIFIED", "APPLY", "WITHDRAW",
		}},
		{paymentv1.RiskExtensionOrigin(0).Descriptor(), "RISK_EXTENSION_ORIGIN_", []string{
			"UNSPECIFIED", "PLATFORM", "SELLER",
		}},
		{paymentv1.RiskExtensionReason(0).Descriptor(), "RISK_EXTENSION_REASON_", []string{
			"UNSPECIFIED", "SELLER_FUNDS_FROZEN", "PAYER_MISMATCH", "REVERSAL_RISK", "UPSTREAM_INVESTIGATION", "PLATFORM_REVIEW", "OTHER",
		}},
		{paymentv1.AmountMismatchSource(0).Descriptor(), "AMOUNT_MISMATCH_SOURCE_", []string{
			"UNSPECIFIED", "QUERY", "CREATE", "CLOSE", "LEGACY",
		}},
		{paymentv1.AmountMismatchKind(0).Descriptor(), "AMOUNT_MISMATCH_KIND_", []string{
			"UNSPECIFIED", "UNKNOWN", "IDENTITY",
		}},
		{paymentv1.AmountMismatchOutcome(0).Descriptor(), "AMOUNT_MISMATCH_OUTCOME_", []string{
			"UNSPECIFIED", "REVERIFY", "REFUND_OBSERVED",
		}},
		{paymentv1.RefundOrigin(0).Descriptor(), "REFUND_ORIGIN_", []string{
			"UNSPECIFIED", "MERCHANT_API", "PLATFORM", "DISPUTE_RULING", "AMOUNT_MISMATCH",
		}},
		{paymentv1.RiskExtensionNoticeKind(0).Descriptor(), "RISK_EXTENSION_NOTICE_KIND_", []string{
			"UNSPECIFIED", "SUBMITTED", "APPLIED", "REJECTED", "EXPIRED",
		}},
		{paymentv1.RiskExtensionState(0).Descriptor(), "RISK_EXTENSION_STATE_", []string{
			"UNSPECIFIED", "HOLDING", "PENDING_APPROVAL", "APPLYING", "APPLIED", "REFUSED", "REJECTED", "WITHDRAWN", "EXPIRED", "SUPERSEDED", "FAILED",
		}},
		{paymentv1.RiskExtensionBlocker(0).Descriptor(), "RISK_EXTENSION_BLOCKER_", []string{
			"UNSPECIFIED", "ORDER_STATUS", "ESCROW_NOT_RESERVED", "LEDGER_NOT_PAID", "LEDGER_DISPUTED", "REFUND_ACTIVE", "EXTENSION_PENDING", "CAP_EXHAUSTED", "LIMIT_REACHED", "SELLER_ALREADY_REQUESTED", "SELLER_PENDING_LIMIT", "LEDGER_VERSION_CHANGED",
		}},
		{paymentv1.AmountMismatchCaseState(0).Descriptor(), "AMOUNT_MISMATCH_CASE_STATE_", []string{
			"UNSPECIFIED", "OPEN", "RECOVERING", "RESOLVED",
		}},
		{paymentv1.AmountMismatchResolution(0).Descriptor(), "AMOUNT_MISMATCH_RESOLUTION_", []string{
			"UNSPECIFIED", "REVERIFIED_PAID", "REVERIFIED_UNPAID", "REFUNDED",
		}},
		{paymentv1.AmountMismatchRecoveryState(0).Descriptor(), "AMOUNT_MISMATCH_RECOVERY_STATE_", []string{
			"UNSPECIFIED", "PENDING_APPROVAL", "EXECUTING", "COMPLETED", "FAILED", "SUPERSEDED", "EXPIRED",
		}},
		{paymentv1.AmountMismatchBlocker(0).Descriptor(), "AMOUNT_MISMATCH_BLOCKER_", []string{
			"UNSPECIFIED", "ORDER_STATUS", "ORDER_STAGE", "ORDER_DISPUTED", "REFUND_ACTIVE", "FORCED_CLOSE_ACTIVE", "ESCROW_NOT_RESERVED", "LEDGER_VERSION_CHANGED", "CASE_NOT_OPEN", "REVERIFY_LIMIT_REACHED", "RECOVERY_PENDING", "STATE_CHANGED",
		}},
	} {
		t.Run(string(tc.enum.Name()), func(t *testing.T) {
			values := tc.enum.Values()
			if values.Len() != len(tc.values) {
				t.Errorf("got %d values, want %d", values.Len(), len(tc.values))
			}
			for number, suffix := range tc.values {
				want := protoreflect.Name(tc.prefix + suffix)
				v := values.ByNumber(protoreflect.EnumNumber(number))
				if v == nil || v.Name() != want {
					t.Errorf("value %d must be %s", number, want)
				}
			}
		})
	}
}

func TestV0280Methods(t *testing.T) {
	reservation := ledgerv1.File_aipay_ledger_v1_reservation_service_proto.Services().ByName("ReservationService")
	risk := paymentv1.File_aipay_payment_v1_risk_extension_proto.Services().ByName("RiskExtensionService")
	seller := paymentv1.File_aipay_payment_v1_seller_portal_service_proto.Services().ByName("SellerPortalService")
	mismatch := paymentv1.File_aipay_payment_v1_amount_mismatch_proto.Services().ByName("AmountMismatchService")
	if risk == nil || risk.Methods().Len() != 6 || mismatch == nil || mismatch.Methods().Len() != 4 {
		t.Fatal("unexpected platform risk extension or amount mismatch RPC surface")
	}
	const (
		serviceOnly = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED
		required    = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED
		recent      = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT
	)
	for _, tc := range []struct {
		service    protoreflect.ServiceDescriptor
		method     protoreflect.Name
		caller     string
		assertion  authzv1.AssertionRequirement
		permission string
		maxAge     uint32
	}{
		{reservation, "HoldTradeForRiskExtension", "payment", serviceOnly, "", 0},
		{reservation, "ResolveTradeRiskExtension", "payment", serviceOnly, "", 0},
		{risk, "GetOrderRiskExtensions", "backend", required, "payment.risk_extension.read", 0},
		{risk, "ListRiskExtensions", "backend", required, "payment.risk_extension.read", 0},
		{risk, "SubmitRiskExtension", "backend", recent, "payment.risk_extension.submit", 600},
		{risk, "ApproveRiskExtension", "backend", required, "payment.risk_extension.approve", 0},
		{risk, "RejectRiskExtension", "backend", required, "payment.risk_extension.approve", 0},
		{risk, "WithdrawRiskExtension", "backend", recent, "payment.risk_extension.submit", 600},
		{seller, "SubmitSellerRiskExtension", "backend", recent, "payment.risk_extension.request", 600},
		{seller, "ListSellerRiskExtensions", "backend", required, "payment.order.read", 0},
		{seller, "WithdrawSellerRiskExtension", "backend", recent, "payment.risk_extension.request", 600},
		{mismatch, "ListAmountMismatchCases", "backend", required, "payment.amount_mismatch.read", 0},
		{mismatch, "GetAmountMismatchCase", "backend", required, "payment.amount_mismatch.read", 0},
		{mismatch, "SubmitAmountMismatchRecovery", "backend", recent, "payment.amount_mismatch.submit", 600},
		{mismatch, "ApproveAmountMismatchRecovery", "backend", required, "payment.amount_mismatch.approve", 0},
	} {
		t.Run(string(tc.method), func(t *testing.T) {
			if tc.service == nil {
				t.Fatal("missing service")
			}
			m := tc.service.Methods().ByName(tc.method)
			name := tc.service.ParentFile().Package().Append(tc.method)
			if m == nil || m.IsStreamingClient() || m.IsStreamingServer() || m.Input().FullName() != name+"Request" || m.Output().FullName() != name+"Response" {
				t.Fatalf("unexpected %s signature", tc.method)
			}
			rule := &authzv1.CallerRule{
				Callers: []string{tc.caller}, Assertion: tc.assertion, RecentAuthMaxAgeSeconds: tc.maxAge,
			}
			if tc.permission != "" {
				rule.Permissions = []string{tc.permission}
			}
			want := &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule}}
			got := proto.GetExtension(m.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
			if !proto.Equal(got, want) {
				t.Errorf("policy = %v, want %v", got, want)
			}
		})
	}
}
