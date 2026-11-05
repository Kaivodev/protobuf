package contracttest

import (
	"slices"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	eventsv1 "github.com/Kaivodev/protobuf/gen/go/aipay/events/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	reconciliationv1 "github.com/Kaivodev/protobuf/gen/go/aipay/reconciliation/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// policycheck validates policy syntax. These checks protect the financial trust
// boundary: payment must not gain an administrative or unilateral release RPC,
// and the event caller union must still include every ledger.trades receiver.
func TestC2CMethodAuthorization(t *testing.T) {
	type expected struct {
		callers    []string
		assertion  authzv1.AssertionRequirement
		permission string
	}
	serviceOnly := authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED
	userRequired := authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED
	services := []struct {
		service protoreflect.ServiceDescriptor
		methods map[string]expected
	}{
		{
			ledgerv1.File_aipay_ledger_v1_reservation_service_proto.Services().ByName("ReservationService"),
			map[string]expected{
				"ReserveTrade":        {[]string{"payment"}, serviceOnly, ""},
				"GetTradeReservation": {[]string{"payment", "reconciliation"}, serviceOnly, ""},
				"HoldTradeForRefund":  {[]string{"payment"}, serviceOnly, ""},
				"HoldTradeForDispute": {[]string{"payment"}, serviceOnly, ""},
				"ReserveRefund":       {[]string{"payment"}, serviceOnly, ""},
			},
		},
		{
			ledgerv1.File_aipay_ledger_v1_control_service_proto.Services().ByName("LedgerControlService"),
			map[string]expected{
				"RequestReadOnlyExit": {[]string{"backend"}, userRequired, "ledger.control.request"},
				"RecordCarryover":     {[]string{"backend"}, userRequired, "ledger.control.request"},
				"ApproveReadOnlyExit": {[]string{"backend"}, userRequired, "ledger.control.approve"},
				"ApproveCarryover":    {[]string{"backend"}, userRequired, "ledger.control.approve"},
			},
		},
		{
			eventsv1.File_aipay_events_v1_event_stream_proto.Services().ByName("EventStreamService"),
			map[string]expected{
				"ListEvents":    {[]string{"ledger", "payment", "reconciliation", "settlement"}, serviceOnly, ""},
				"GetStreamHead": {[]string{"ledger", "payment", "reconciliation", "settlement"}, serviceOnly, ""},
			},
		},
	}
	for _, group := range services {
		if group.service == nil {
			t.Fatal("required service missing")
		}
		if group.service.Methods().Len() != len(group.methods) {
			t.Fatalf("%s exposes an unreviewed RPC set", group.service.FullName())
		}
		for name, want := range group.methods {
			t.Run(string(group.service.Name())+"/"+name, func(t *testing.T) {
				method := group.service.Methods().ByName(protoreflect.Name(name))
				if method == nil || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
					t.Fatal("missing method or authorization policy")
				}
				policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
				if len(policy.GetRules()) != 1 {
					t.Fatalf("unexpected caller rules: %v", policy)
				}
				rule := policy.GetRules()[0]
				callers := slices.Sorted(slices.Values(rule.GetCallers()))
				if !slices.Equal(callers, want.callers) || rule.GetAssertion() != want.assertion || rule.GetRecentAuthMaxAgeSeconds() != 0 {
					t.Fatalf("unexpected policy: %v", rule)
				}
				var permissions []string
				if want.permission != "" {
					permissions = []string{want.permission}
				}
				if !slices.Equal(rule.GetPermissions(), permissions) {
					t.Fatalf("permissions %v, want %v", rule.GetPermissions(), permissions)
				}
			})
		}
	}
}

// Every existing authorization must carry the same presence-aware ruling
// evidence. Its closed shape must not introduce an account or amount override.
func TestC2CDisputeResolutionContract(t *testing.T) {
	ruling := (&reconciliationv1.DisputeResolution{}).ProtoReflect().Descriptor()
	wantFields := []struct {
		name protoreflect.Name
		kind protoreflect.Kind
	}{
		{"dispute_id", protoreflect.StringKind},
		{"ruling_id", protoreflect.StringKind},
		{"ruling_sha256", protoreflect.BytesKind},
		{"approved_at", protoreflect.MessageKind},
	}
	if ruling.Fields().Len() != len(wantFields) {
		t.Fatalf("%s must contain only the four reviewed evidence fields", ruling.FullName())
	}
	for i, want := range wantFields {
		field := ruling.Fields().ByName(want.name)
		if field == nil || field.Number() != protoreflect.FieldNumber(i+1) || field.Kind() != want.kind || field.Cardinality() != protoreflect.Optional || field.ContainingOneof() != nil {
			t.Fatalf("unexpected ruling field %s: %v", want.name, field)
		}
	}
	approvedAt := ruling.Fields().ByName("approved_at")
	if !approvedAt.HasPresence() || approvedAt.Message().FullName() != "google.protobuf.Timestamp" {
		t.Fatal("approved_at must be a presence-aware Timestamp")
	}

	for _, event := range []struct {
		message proto.Message
		number  protoreflect.FieldNumber
	}{
		{&reconciliationv1.TradeReleaseAuthorized{}, 9},
		{&reconciliationv1.TradeCancellationAuthorized{}, 8},
		{&reconciliationv1.TradeRefundAuthorized{}, 12},
	} {
		t.Run(string(event.message.ProtoReflect().Descriptor().Name()), func(t *testing.T) {
			field := event.message.ProtoReflect().Descriptor().Fields().ByName("dispute_resolution")
			if field == nil || field.Number() != event.number || field.Kind() != protoreflect.MessageKind || field.Cardinality() != protoreflect.Optional || !field.HasPresence() || field.ContainingOneof() != nil {
				t.Fatalf("missing optional ruling at field %d: %v", event.number, field)
			}
			if field.Message().FullName() != ruling.FullName() {
				t.Fatalf("ruling type %s, want %s", field.Message().FullName(), ruling.FullName())
			}
		})
	}
}
