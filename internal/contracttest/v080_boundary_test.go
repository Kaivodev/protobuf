package contracttest

import (
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	commonv1 "github.com/Kaivodev/protobuf/gen/go/aipay/common/v1"
	iamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/iam/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	reconciliationv1 "github.com/Kaivodev/protobuf/gen/go/aipay/reconciliation/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Differentiate explicit disabled/unusable from omission. No transport may silently turn false into true.
func TestV080ExplicitDisabledPresence(t *testing.T) {
	for _, msg := range []proto.Message{
		&iamv1.Application{Enabled: proto.Bool(false), AllowEpayV1: proto.Bool(false), AllowEpayV2: proto.Bool(true)},
		&iamv1.CreateApplicationRequest{AllowEpayV1: proto.Bool(false), AllowEpayV2: proto.Bool(true)},
		&iamv1.SetApplicationEnabledRequest{Enabled: proto.Bool(false)},
		&iamv1.ApplicationOwnership{Enabled: proto.Bool(false)},
		&reconciliationv1.VerificationAccount{Enabled: proto.Bool(false)},
		&paymentv1.RateSnapshotRecorded{UsableForQuoting: proto.Bool(false)},
	} {
		t.Run(string(msg.ProtoReflect().Descriptor().Name()), func(t *testing.T) {
			raw, err := proto.Marshal(msg)
			if err != nil {
				t.Fatal(err)
			}
			decoded := msg.ProtoReflect().Type().New().Interface()
			if err := proto.Unmarshal(raw, decoded); err != nil {
				t.Fatal(err)
			}
			msg.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
				if !field.HasPresence() || !decoded.ProtoReflect().Has(field) || decoded.ProtoReflect().Get(field).Interface() != value.Interface() || msg.ProtoReflect().Type().New().Has(field) {
					t.Errorf("lost presence for %s", field.FullName())
				}
				return true
			})
		})
	}
}

func requireField(t *testing.T, msg proto.Message, name protoreflect.Name, number protoreflect.FieldNumber, kind protoreflect.Kind, typ protoreflect.FullName) protoreflect.FieldDescriptor {
	t.Helper()
	field := msg.ProtoReflect().Descriptor().Fields().ByName(name)
	if field == nil || field.Number() != number || field.Kind() != kind {
		t.Fatalf("missing/changed field %s.%s", msg.ProtoReflect().Descriptor().FullName(), name)
	}
	if typ != "" && (kind != protoreflect.MessageKind || field.Message().FullName() != typ) {
		t.Fatalf("wrong message type for %s", field.FullName())
	}
	return field
}

// Entry binding is additive to the published token, not another credential
// alternative; challenge answers are mutually exclusive. A reauthentication
// carries a concrete request/scope/resource, never an unscoped upgrade flag.
func TestV080WebIdentityBoundary(t *testing.T) {
	requireField(t, &iamv1.BearerCredential{}, "token", 1, protoreflect.StringKind, "")
	requireField(t, &iamv1.BearerCredential{}, "entry", 2, protoreflect.EnumKind, "")
	requireField(t, &iamv1.BearerCredential{}, "reauthentication_id", 3, protoreflect.StringKind, "")
	requireField(t, &iamv1.RequestBinding{}, "body_sha256", 4, protoreflect.BytesKind, "")
	for _, want := range []struct {
		msg   proto.Message
		oneof protoreflect.Name
		names []protoreflect.Name
	}{
		{&iamv1.AuthenticationResult{}, "next", []protoreflect.Name{"session", "challenge"}},
		{&iamv1.VerifyChallengeRequest{}, "answer", []protoreflect.Name{"totp_code", "webauthn"}},
		{&iamv1.AcceptInvitationRequest{}, "acceptance", []protoreflect.Name{"new_user", "existing_session"}},
	} {
		oneof := want.msg.ProtoReflect().Descriptor().Oneofs().ByName(want.oneof)
		if oneof == nil || oneof.Fields().Len() != len(want.names) {
			t.Fatal("unreviewed bootstrap branch")
		}
		for _, name := range want.names {
			if oneof.Fields().ByName(name) == nil {
				t.Fatalf("missing %s", name)
			}
		}
	}
	for _, want := range []struct {
		name protoreflect.Name
		typ  protoreflect.FullName
	}{
		{"binding", "aipay.iam.v1.RequestBinding"}, {"acting_scope", "aipay.common.v1.Scope"}, {"resource", "aipay.common.v1.ResourceRef"},
	} {
		field := (&iamv1.ReauthenticationTarget{}).ProtoReflect().Descriptor().Fields().ByName(want.name)
		if field == nil || field.Kind() != protoreflect.MessageKind || !field.HasPresence() || field.Message().FullName() != want.typ {
			t.Fatalf("missing reauthentication binding %s", want.name)
		}
	}
	// Leaf enum prevents cyclic identity imports and leaves future merchant web
	// support additive without silently accepting another current entry.
	entries := iamv1.WebEntry(0).Descriptor().Values()
	for i, name := range []protoreflect.Name{"WEB_ENTRY_UNSPECIFIED", "WEB_ENTRY_PLATFORM_ADMIN", "WEB_ENTRY_UPSTREAM_PORTAL"} {
		if entries.Len() != 3 || entries.Get(i).Name() != name || entries.Get(i).Number() != protoreflect.EnumNumber(i) {
			t.Fatal("unreviewed web entry")
		}
	}
	for _, msg := range []proto.Message{&iamv1.SessionInfo{}, &iamv1.UserProfile{}, &iamv1.Application{}, &iamv1.ApplicationCredential{}, &iamv1.MfaMethod{}, &iamv1.Invitation{}} {
		fields := msg.ProtoReflect().Descriptor().Fields()
		for _, name := range []protoreflect.Name{"token", "key", "password", "secret", "recovery_codes", "secret_ciphertext"} {
			if fields.ByName(name) != nil {
				t.Fatalf("secret added to ordinary read model: %s.%s", msg.ProtoReflect().Descriptor().FullName(), name)
			}
		}
	}
	// Browser bootstrap enrollment must use the authenticated session from proof,
	// not a bot page token, caller user_id, or the generic UserAuthentication union.
	for _, msg := range []proto.Message{&iamv1.BeginSessionTotpEnrollmentRequest{}, &iamv1.ConfirmSessionTotpEnrollmentRequest{}, &iamv1.BeginWebAuthnRegistrationRequest{}, &iamv1.CompleteWebAuthnRegistrationRequest{}} {
		fields := msg.ProtoReflect().Descriptor().Fields()
		for _, name := range []protoreflect.Name{"token", "user_id", "authentication", "telegram"} {
			if fields.ByName(name) != nil {
				t.Fatalf("wrong session enrollment authority: %s", name)
			}
		}
	}
}

// Reconciliation owns the approval union and plugin/account materials. It may
// reuse wire WebAuthn evidence, never payment's material or credential storage.
func TestV080ReconciliationDomainBoundary(t *testing.T) {
	for _, file := range []protoreflect.FileDescriptor{
		reconciliationv1.File_aipay_reconciliation_v1_admin_approval_proto,
		reconciliationv1.File_aipay_reconciliation_v1_reconciliation_admin_service_proto,
	} {
		for i := 0; i < file.Imports().Len(); i++ {
			if file.Imports().Get(i).Package() == "aipay.payment.v1" {
				t.Fatal("reconciliation depends on payment approval material")
			}
		}
	}
	material := (&reconciliationv1.ReconciliationApprovalMaterial{}).ProtoReflect().Descriptor()
	operation := material.Oneofs().ByName("operation")
	if operation == nil || operation.Fields().Len() != 3 {
		t.Fatal("unreviewed approval union")
	}
	for i, name := range []protoreflect.Name{"verification_account", "plugin_version", "plugin_enable"} {
		field := operation.Fields().ByName(name)
		if field == nil || field.Number() != protoreflect.FieldNumber(6+i) || field.Message().ParentFile().Package() != "aipay.reconciliation.v1" {
			t.Fatalf("wrong material branch %s", name)
		}
	}
	for _, want := range []struct {
		name   protoreflect.Name
		number protoreflect.FieldNumber
		typ    protoreflect.FullName
	}{
		{"approval_request", 1, "aipay.approval.v1.ApprovalRequest"}, {"material", 2, "aipay.reconciliation.v1.ReconciliationApprovalMaterial"}, {"approvals", 3, "aipay.approval.v1.Approval"},
	} {
		requireField(t, &reconciliationv1.ReconciliationApproval{}, want.name, want.number, protoreflect.MessageKind, want.typ)
	}
	// The package is bound by digest; bytes travel only in UploadPluginRequest.
	requireField(t, &reconciliationv1.PluginVersionMaterial{}, "fixtures_sha256", 3, protoreflect.BytesKind, "")
	if reconciliationv1.File_aipay_reconciliation_v1_admin_approval_proto.Messages().ByName("PluginVersionMaterial").Fields().ByName("plugin_wasm") != nil {
		t.Fatal("approval material must not carry plugin bytes")
	}
	requireField(t, &paymentv1.PluginVersionMaterial{}, "fixtures_sha256", 8, protoreflect.BytesKind, "")
	requireField(t, &reconciliationv1.PluginVersionMaterial{}, "package_sha256", 5, protoreflect.BytesKind, "")
	requireField(t, &reconciliationv1.PluginVersionMaterial{}, "regression_report_sha256", 6, protoreflect.BytesKind, "")
	requireField(t, &reconciliationv1.PluginEnableChange{}, "publication_approval_request_id", 5, protoreflect.StringKind, "")
	requireField(t, &reconciliationv1.PluginEnableChange{}, "publication_request_digest", 6, protoreflect.BytesKind, "")
	for _, msg := range []proto.Message{&reconciliationv1.VerificationAccountConfiguration{}, &reconciliationv1.StoreVerificationCredentialsResponse{}} {
		require := msg.ProtoReflect().Descriptor().Fields().ByName("credentials_ref")
		if require == nil || require.Kind() != protoreflect.StringKind {
			t.Fatal("missing encrypted domain-local reference")
		}
		for _, name := range []protoreflect.Name{"credentials_json", "secret", "key"} {
			if msg.ProtoReflect().Descriptor().Fields().ByName(name) != nil {
				t.Fatal("secret escaped credential ingress")
			}
		}
	}
	for _, msg := range []proto.Message{&reconciliationv1.ApprovePluginRequest{}, &reconciliationv1.ApprovePluginEnableRequest{}, &reconciliationv1.ApproveVerificationAccountRequest{}} {
		requireField(t, msg, "assertion", 3, protoreflect.MessageKind, "aipay.approval.v1.WebAuthnAssertion")
	}
}

// Quotes still identify rates by the published field 10. New signed facts
// supply the minute and per-venue evidence without rewriting locked quotes.
func TestV080SignedFactShape(t *testing.T) {
	requireField(t, &commonv1.TradeQuoteSnapshot{}, "rate_snapshot_id", 10, protoreflect.StringKind, "")
	if (&commonv1.TradeQuoteSnapshot{}).ProtoReflect().Descriptor().Fields().Len() != 19 {
		t.Fatal("quote snapshot unexpectedly changed")
	}
	for _, name := range []protoreflect.Name{"minute", "collected_at", "expires_at"} {
		field := (&paymentv1.RateSnapshotRecorded{}).ProtoReflect().Descriptor().Fields().ByName(name)
		if field == nil || field.Kind() != protoreflect.MessageKind || field.Message().FullName() != "google.protobuf.Timestamp" {
			t.Fatalf("missing typed rate time %s", name)
		}
	}
	for _, name := range []protoreflect.Name{"okx_median", "binance_median"} {
		field := (&paymentv1.RateSnapshotRecorded{}).ProtoReflect().Descriptor().Fields().ByName(name)
		if field == nil || field.Kind() != protoreflect.MessageKind || field.Message().FullName() != "aipay.common.v1.ExchangeRate" {
			t.Fatalf("inexact rate %s", name)
		}
	}
	for _, name := range []protoreflect.Name{"okx_sample_count", "binance_sample_count"} {
		field := (&paymentv1.RateSnapshotRecorded{}).ProtoReflect().Descriptor().Fields().ByName(name)
		if field == nil || field.Kind() != protoreflect.Uint32Kind {
			t.Fatalf("missing source count %s", name)
		}
	}
	for _, msg := range []proto.Message{&iamv1.ApplicationCreated{}, &iamv1.ApplicationStatusChanged{}} {
		requireField(t, msg, "ownership", 1, protoreflect.MessageKind, "aipay.iam.v1.ApplicationOwnership")
	}
	fields := (&iamv1.ApplicationOwnership{}).ProtoReflect().Descriptor().Fields()
	if fields.Len() != 7 {
		t.Fatal("unreviewed ownership fact fields")
	}
	for _, name := range []protoreflect.Name{"account_id", "business_id", "application_id", "pid", "ownership_version", "application_version", "enabled"} {
		if fields.ByName(name) == nil {
			t.Fatalf("missing ownership fact %s", name)
		}
	}
}

// A channel selector from backend is not a trusted mapping. Independent
// verification gets only the approved identity, never payment's secret store.
func TestV080ChannelVerificationMapping(t *testing.T) {
	service := paymentv1.File_aipay_payment_v1_channel_verification_service_proto.Services().ByName("ChannelVerificationService")
	if service == nil || service.Methods().Len() != 1 {
		t.Fatal("unreviewed channel verification surface")
	}
	method := service.Methods().ByName("GetChannelVerificationBinding")
	if method == nil || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
		t.Fatal("missing mapping read policy")
	}
	policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
	if len(policy.GetRules()) != 1 {
		t.Fatal("unexpected caller rules")
	}
	rule := policy.GetRules()[0]
	if len(rule.GetCallers()) != 1 || rule.GetCallers()[0] != "reconciliation" || rule.GetAssertion() != authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED || len(rule.GetPermissions()) != 0 || rule.GetRecentAuthMaxAgeSeconds() != 0 {
		t.Fatalf("unexpected mapping read policy: %v", rule)
	}
	if method.IsStreamingClient() || method.IsStreamingServer() {
		t.Fatal("mapping read must remain unary")
	}
	fields := (&paymentv1.ChannelVerificationBinding{}).ProtoReflect().Descriptor().Fields()
	if fields.Len() != 3 {
		t.Fatal("mapping response must not expose payment secrets/configuration")
	}
	requireField(t, &paymentv1.ChannelVerificationBinding{}, "channel_account_id", 1, protoreflect.StringKind, "")
	requireField(t, &paymentv1.ChannelVerificationBinding{}, "channel_code", 2, protoreflect.StringKind, "")
	requireField(t, &paymentv1.ChannelVerificationBinding{}, "control_version", 3, protoreflect.Uint64Kind, "")
}
