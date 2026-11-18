package contracttest

import (
	"testing"

	iamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/iam/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0110FreezeSnapshot(t *testing.T) {
	req := &ledgerv1.GetTradeFreezeDetailsRequest{}
	details := &ledgerv1.GetTradeFreezeDetailsResponse{}
	requireField(t, req, "order_id", 1, protoreflect.StringKind, "")
	requireField(t, req, "escrow_reservation_id", 2, protoreflect.StringKind, "")
	requireField(t, details, "as_of_seq", 1, protoreflect.Uint64Kind, "")
	requireField(t, details, "order_id", 2, protoreflect.StringKind, "")
	requireField(t, details, "escrow_reservation_id", 3, protoreflect.StringKind, "")
	status := requireField(t, details, "status", 4, protoreflect.EnumKind, "")
	if status.Enum().FullName() != "aipay.ledger.v1.TradeReservationStatus" {
		t.Fatal("unknown escrow status type")
	}
	requireField(t, details, "control_version", 5, protoreflect.Uint64Kind, "")
	refunds := requireField(t, details, "refund_holds", 6, protoreflect.MessageKind, "aipay.ledger.v1.TradeRefundHold")
	disputes := requireField(t, details, "dispute_holds", 7, protoreflect.MessageKind, "aipay.ledger.v1.TradeDisputeHold")
	if !refunds.IsList() || !disputes.IsList() {
		t.Fatal("snapshot must carry every open barrier")
	}

	refund := &ledgerv1.TradeRefundHold{}
	dispute := &ledgerv1.TradeDisputeHold{}
	requireField(t, refund, "refund_id", 1, protoreflect.StringKind, "")
	requireField(t, dispute, "dispute_id", 1, protoreflect.StringKind, "")
	for _, msg := range []proto.Message{refund, dispute} {
		requireField(t, msg, "control_version", 2, protoreflect.Uint64Kind, "")
	}
	requireField(t, refund, "reservation", 3, protoreflect.MessageKind, "aipay.ledger.v1.RefundReservation")
	requireField(t, dispute, "released_hold", 3, protoreflect.MessageKind, "aipay.ledger.v1.ReleasedTradeDisputeHold")
	requireField(t, details, "seller_risk_freeze", 8, protoreflect.MessageKind, "aipay.ledger.v1.AccountRiskFreeze")
	requireField(t, details, "recipient_risk_freeze", 9, protoreflect.MessageKind, "aipay.ledger.v1.AccountRiskFreeze")
	risk := &ledgerv1.AccountRiskFreeze{}
	requireField(t, risk, "account_id", 1, protoreflect.StringKind, "")
	frozen := requireField(t, risk, "frozen", 2, protoreflect.BoolKind, "")
	ids := requireField(t, risk, "freeze_ids", 3, protoreflect.StringKind, "")
	if !frozen.HasOptionalKeyword() || !ids.IsList() {
		t.Fatal("account requires explicit frozen state and all active freeze IDs")
	}
}

func TestV0110CallbackPolicy(t *testing.T) {
	requireField(t, &paymentv1.GetCallbackPolicyRequest{}, "channel_code", 1, protoreflect.StringKind, "")
	policy := &paymentv1.GetCallbackPolicyResponse{}
	for i, name := range []protoreflect.Name{"source_ip_cidrs", "allowed_methods", "allowed_content_types"} {
		if !requireField(t, policy, name, protoreflect.FieldNumber(i+1), protoreflect.StringKind, "").IsList() {
			t.Fatalf("%s must be a list", name)
		}
	}
	requireField(t, policy, "allow_query", 4, protoreflect.BoolKind, "")
	requireField(t, policy, "max_body_bytes", 5, protoreflect.Uint32Kind, "")
	requireField(t, policy, "policy_version", 6, protoreflect.Uint64Kind, "")
	if !requireField(t, &paymentv1.ChannelAccountConfiguration{}, "callback_source_cidrs", 16, protoreflect.StringKind, "").IsList() {
		t.Fatal("channel approval must carry the CIDR list")
	}
}

func TestV0110WebIdentity(t *testing.T) {
	requireField(t, &iamv1.AuthorizeRequest{}, "request_binding", 8, protoreflect.MessageKind, "aipay.iam.v1.RequestBinding")
	for _, tc := range []struct {
		msg    proto.Message
		number protoreflect.FieldNumber
	}{{&iamv1.RequestBinding{}, 5}, {&iamv1.LoginRequest{}, 6}} {
		if !requireField(t, tc.msg, "client_ip", tc.number, protoreflect.StringKind, "").HasOptionalKeyword() {
			t.Fatal("missing optional trusted client IP")
		}
	}
	opts := &iamv1.GetChallengeResponse{}
	if opts.ProtoReflect().Descriptor().Fields().Len() != 4 {
		t.Fatal("GetChallenge must return only purpose, expiry and options")
	}
	req := &iamv1.GetChallengeRequest{}
	requireField(t, req, "challenge_id", 1, protoreflect.StringKind, "")
	requireField(t, req, "challenge_token", 2, protoreflect.StringKind, "")
	requireField(t, req, "entry", 3, protoreflect.EnumKind, "")
	requireField(t, req, "preauthentication_context", 4, protoreflect.BytesKind, "")
	for _, tc := range []struct {
		msg   proto.Message
		union protoreflect.Name
		names []protoreflect.Name
	}{
		{opts, "options", []protoreflect.Name{"webauthn_options_json", "totp"}},
		{&iamv1.VerifyChallengeRequest{}, "answer", []protoreflect.Name{"totp_code", "webauthn", "recovery_code"}},
		{&iamv1.BeginTotpReplacementRequest{}, "current_factor", []protoreflect.Name{"current_totp_code", "recovery_code"}},
	} {
		union := tc.msg.ProtoReflect().Descriptor().Oneofs().ByName(tc.union)
		if union == nil || union.Fields().Len() != len(tc.names) {
			t.Fatalf("unexpected %s branches", tc.union)
		}
		for _, name := range tc.names {
			if union.Fields().ByName(name) == nil {
				t.Fatalf("missing %s.%s", tc.union, name)
			}
		}
	}
	requireField(t, &iamv1.ConfirmTotpReplacementRequest{}, "new_totp_code", 3, protoreflect.StringKind, "")
	count := requireField(t, &iamv1.VerifyChallengeResponse{}, "remaining_recovery_codes", 2, protoreflect.Uint32Kind, "")
	if !count.HasOptionalKeyword() {
		t.Fatal("last code spent must be distinguishable from ordinary login")
	}
	if (&iamv1.ListInvitationsRequest{}).ProtoReflect().Descriptor().Fields().ByName("email") != nil {
		t.Fatal("invitation list must not expose an email filter")
	}
}

// Zero funded holds, explicit clear states and the count after spending the
// last recovery code must survive transport even when their scalars are zero.
func TestV0110ZeroValueEvidenceRoundTrip(t *testing.T) {
	for _, msg := range []proto.Message{
		&ledgerv1.TradeRefundHold{RefundId: "refund", ControlVersion: 2},
		&ledgerv1.GetTradeFreezeDetailsResponse{
			AsOfSeq: 2, OrderId: "order", EscrowReservationId: "escrow",
			Status: ledgerv1.TradeReservationStatus_TRADE_RESERVATION_STATUS_RELEASED, ControlVersion: 3,
			DisputeHolds: []*ledgerv1.TradeDisputeHold{{
				DisputeId: "d", ControlVersion: 3,
				ReleasedHold: &ledgerv1.ReleasedTradeDisputeHold{DisputeId: "d", ShortfallUsdt: 1},
			}},
			SellerRiskFreeze:    &ledgerv1.AccountRiskFreeze{AccountId: "s", Frozen: proto.Bool(false)},
			RecipientRiskFreeze: &ledgerv1.AccountRiskFreeze{AccountId: "p", Frozen: proto.Bool(true), FreezeIds: []string{"f"}},
		},
		&iamv1.VerifyChallengeResponse{RemainingRecoveryCodes: proto.Uint32(0)},
	} {
		raw, err := proto.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}
		got := msg.ProtoReflect().Type().New().Interface()
		if err := proto.Unmarshal(raw, got); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(msg, got) {
			t.Fatalf("lost field presence in %s", msg.ProtoReflect().Descriptor().FullName())
		}
	}
}
