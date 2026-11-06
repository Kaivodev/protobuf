package contracttest

import (
	"slices"
	"strings"
	"testing"

	commonv1 "github.com/Kaivodev/protobuf/gen/go/aipay/common/v1"
	iamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/iam/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	reconciliationv1 "github.com/Kaivodev/protobuf/gen/go/aipay/reconciliation/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func walkV090Fields(t *testing.T, msg protoreflect.MessageDescriptor, check func(protoreflect.FieldDescriptor)) {
	t.Helper()
	seen := map[protoreflect.FullName]bool{}
	var visit func(protoreflect.MessageDescriptor)
	visit = func(desc protoreflect.MessageDescriptor) {
		if seen[desc.FullName()] {
			return
		}
		seen[desc.FullName()] = true
		for i := 0; i < desc.Fields().Len(); i++ {
			field := desc.Fields().Get(i)
			check(field)
			if field.Message() != nil {
				visit(field.Message())
			}
		}
	}
	visit(msg)
}

func TestV090SellerAuthorityAndProjection(t *testing.T) {
	for _, group := range v090Groups() {
		for name := range group.methods {
			if !v090SellerMethod(group.service, name) {
				continue
			}
			method := group.file.Services().ByName(group.service).Methods().ByName(name)
			t.Run(string(method.FullName()), func(t *testing.T) {
				walkV090Fields(t, method.Input(), func(field protoreflect.FieldDescriptor) {
					if strings.Contains(string(field.Name()), "account_id") && field.Name() != "channel_account_id" || slices.Contains([]protoreflect.Name{"scope", "acting_scope", "owner", "user_id", "principal", "authentication"}, field.Name()) || field.Message() != nil && field.Message().FullName() == "aipay.common.v1.Scope" {
						t.Errorf("self-asserted seller authority: %s", field.FullName())
					}
				})
				walkV090Fields(t, method.Output(), func(field protoreflect.FieldDescriptor) {
					if slices.Contains([]protoreflect.Name{"account_id", "application_id", "credentials_ref", "config_json", "notification", "recipient_evidence", "request_bytes", "assertion"}, field.Name()) {
						t.Errorf("private counterparty/platform field: %s", field.FullName())
					}
					if field.Message() != nil && slices.Contains([]protoreflect.FullName{
						"aipay.payment.v1.Order", "aipay.payment.v1.Refund", "aipay.common.v1.TradeSnapshot", "aipay.common.v1.TradeQuoteSnapshot",
						"aipay.payment.v1.OrderNotificationSnapshot", "aipay.payment.v1.PaymentApproval", "aipay.payment.v1.DisputeEvidence",
						"aipay.ledger.v1.RefundReservation", "aipay.ledger.v1.TradeReservation", "aipay.approval.v1.Approval", "aipay.approval.v1.WebAuthnAssertion",
					}, field.Message().FullName()) {
						t.Errorf("unsafe nested projection: %s", field.FullName())
					}
				})
			})
		}
	}
}

func TestV090PaginationAndReadSnapshots(t *testing.T) {
	lists := 0
	for _, group := range v090Groups() {
		for name := range group.methods {
			method := group.file.Services().ByName(group.service).Methods().ByName(name)
			if strings.HasPrefix(string(name), "List") {
				lists++
				limit := method.Input().Fields().ByName("limit")
				token := method.Input().Fields().ByName("page_token")
				next := method.Output().Fields().ByName("next_page_token")
				if limit == nil || limit.Kind() != protoreflect.Uint32Kind || !proto.HasExtension(limit.Options(), commonv1.E_MaxItems) || proto.GetExtension(limit.Options(), commonv1.E_MaxItems).(uint32) != 100 {
					t.Fatalf("%s lost explicit page bound 100", method.FullName())
				}
				if token == nil || token.Kind() != protoreflect.StringKind || next == nil || next.Kind() != protoreflect.StringKind {
					t.Fatalf("%s lost opaque cursor", method.FullName())
				}
			}
			if name == "EmergencyDisablePlugin" || name == "SignMerchantNotification" {
				continue
			}
			asOf := method.Output().Fields().ByName("as_of")
			if asOf == nil || asOf.Message() == nil || asOf.Message().FullName() != "google.protobuf.Timestamp" {
				t.Fatalf("%s lost read snapshot time", method.FullName())
			}
			if group.service == "LedgerQueryService" {
				seq := method.Output().Fields().ByName("as_of_seq")
				if seq == nil || seq.Kind() != protoreflect.Uint64Kind {
					t.Fatalf("%s lost journal snapshot sequence", method.FullName())
				}
			}
		}
	}
	if lists != 20 {
		t.Fatalf("unreviewed pagination surface: %d", lists)
	}
	batch := requireField(t, &ledgerv1.BatchGetSellerInventoryRequest{}, "seller_account_ids", 1, protoreflect.StringKind, "")
	if !batch.IsList() || !proto.HasExtension(batch.Options(), commonv1.E_MaxItems) || proto.GetExtension(batch.Options(), commonv1.E_MaxItems).(uint32) != 100 {
		t.Fatal("inventory batch lost bound 100")
	}
}

func TestV090EmergencyDisableAndResumeFence(t *testing.T) {
	for _, request := range []proto.Message{&paymentv1.EmergencyDisablePluginRequest{}, &reconciliationv1.EmergencyDisablePluginRequest{}} {
		requireField(t, request, "idempotency_key", 1, protoreflect.StringKind, "")
		requireField(t, request, "plugin_version_id", 2, protoreflect.StringKind, "")
		requireField(t, request, "expected_control_version", 3, protoreflect.Uint64Kind, "")
		requireField(t, request, "reason", 4, protoreflect.StringKind, "")
		requireField(t, request, "assertion", 5, protoreflect.MessageKind, "aipay.approval.v1.WebAuthnAssertion")
		if request.ProtoReflect().Descriptor().Fields().Len() != 5 {
			t.Fatal("unreviewed emergency command")
		}
	}
	for _, response := range []proto.Message{&paymentv1.EmergencyDisablePluginResponse{}, &reconciliationv1.EmergencyDisablePluginResponse{}} {
		requireField(t, response, "control_version", 2, protoreflect.Uint64Kind, "")
		field := response.ProtoReflect().Descriptor().Fields().ByName("affected_accounts")
		if field == nil || !field.IsList() || field.Message().Fields().ByName("control_version").Kind() != protoreflect.Uint64Kind {
			t.Fatal("missing complete affected account projection")
		}
	}
	for _, want := range []struct {
		message proto.Message
		number  protoreflect.FieldNumber
	}{
		{&paymentv1.RequestPluginEnableRequest{}, 5}, {&reconciliationv1.RequestPluginEnableRequest{}, 5},
		{&paymentv1.PluginEnableChange{}, 7}, {&reconciliationv1.PluginEnableChange{}, 7},
	} {
		field := requireField(t, want.message, "expected_plugin_control_version", want.number, protoreflect.Uint64Kind, "")
		if !field.HasOptionalKeyword() {
			t.Fatal("resume fence must distinguish legacy absence from explicit version")
		}
	}
}

func TestV090LedgerLiabilityAndEvidenceShape(t *testing.T) {
	balance := (&ledgerv1.SellerBalance{}).ProtoReflect().Descriptor().Fields()
	if balance.Len() != 7 {
		t.Fatal("unreviewed liability category")
	}
	for i, name := range []protoreflect.Name{"available_usdt", "order_escrow_usdt", "risk_held_usdt", "refund_reserved_usdt", "payout_in_flight_usdt"} {
		requireField(t, &ledgerv1.SellerBalance{}, name, protoreflect.FieldNumber(i+3), protoreflect.Int64Kind, "")
	}
	requireField(t, &paymentv1.AppendSellerEvidenceRequest{}, "expected_evidence_version", 3, protoreflect.Uint64Kind, "")
	requireField(t, &paymentv1.AppendSellerEvidenceRequest{}, "seller_evidence", 4, protoreflect.BytesKind, "")
	requireField(t, &paymentv1.SellerDisputeView{}, "evidence_version", 10, protoreflect.Uint64Kind, "")
	for _, request := range []proto.Message{&paymentv1.ListApprovalsRequest{}, &reconciliationv1.ListApprovalsRequest{}} {
		if request.ProtoReflect().Descriptor().Fields().ByName("owner_service") != nil {
			t.Fatal("approval aggregation belongs to backend, not an owner selector")
		}
	}
}

func TestV090NotificationSigningWhitelist(t *testing.T) {
	params := (&iamv1.MerchantNotificationParameters{}).ProtoReflect().Descriptor().Fields()
	want := []protoreflect.Name{"pid", "trade_no", "out_trade_no", "type", "name", "money", "trade_status", "param"}
	if params.Len() != len(want) {
		t.Fatal("unreviewed notification signing field")
	}
	for i, name := range want {
		requireField(t, &iamv1.MerchantNotificationParameters{}, name, protoreflect.FieldNumber(i+1), protoreflect.StringKind, "")
	}
	request := &iamv1.SignMerchantNotificationRequest{}
	requireField(t, request, "application_id", 1, protoreflect.StringKind, "")
	requireField(t, request, "notification_id", 2, protoreflect.StringKind, "")
	requireField(t, request, "order_id", 3, protoreflect.StringKind, "")
	requireField(t, request, "credential_version", 4, protoreflect.StringKind, "")
	requireField(t, request, "parameters", 5, protoreflect.MessageKind, "aipay.iam.v1.MerchantNotificationParameters")
	if request.ProtoReflect().Descriptor().Fields().Len() != 5 {
		t.Fatal("unreviewed generic signing input")
	}
	if (&iamv1.SignMerchantNotificationResponse{}).ProtoReflect().Descriptor().Fields().Len() != 2 {
		t.Fatal("signing result must contain only sign/sign_type")
	}
}

func TestV090ReconciliationReadMetadata(t *testing.T) {
	for _, want := range []struct {
		message proto.Message
		number  protoreflect.FieldNumber
	}{
		{&reconciliationv1.ListVerificationAccountsResponse{}, 3},
		{&reconciliationv1.GetVerificationAccountResponse{}, 2},
		{&reconciliationv1.ListPluginsResponse{}, 3},
		{&reconciliationv1.GetPluginResponse{}, 3},
		{&reconciliationv1.GetApprovalResponse{}, 2},
	} {
		requireField(t, want.message, "as_of", want.number, protoreflect.MessageKind, "google.protobuf.Timestamp")
	}
	requireField(t, &reconciliationv1.PluginVersion{}, "control_version", 8, protoreflect.Uint64Kind, "")
	disabled := requireField(t, &reconciliationv1.PluginVersion{}, "emergency_disabled", 9, protoreflect.BoolKind, "")
	if !disabled.HasOptionalKeyword() {
		t.Fatal("disabled must have presence")
	}
}
