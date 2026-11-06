package contracttest

import (
	"slices"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	iamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/iam/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// IAM checks body credentials itself; a bot service identity, permission rule
// or metadata-assertion requirement must not silently replace that boundary.
func TestIAMMethodAuthorization(t *testing.T) {
	for _, want := range []struct {
		file    protoreflect.FileDescriptor
		service protoreflect.Name
		methods []protoreflect.Name
	}{
		{iamv1.File_aipay_iam_v1_authorization_proto, "AuthorizationService", []protoreflect.Name{"Authorize"}},
		{iamv1.File_aipay_iam_v1_telegram_service_proto, "TelegramService", []protoreflect.Name{
			"Register", "CreateBindingCode", "Bind", "Unbind", "BeginGroupBinding", "ConfirmGroupBinding", "InvalidateGroupBinding", "GetGroupBinding",
		}},
		{iamv1.File_aipay_iam_v1_mfa_service_proto, "MfaService", []protoreflect.Name{
			"IssueTotpEnrollmentToken", "GetTotpEnrollment", "CompleteTotpEnrollment", "UseRecoveryCode", "ResetTotp",
		}},
	} {
		t.Run(string(want.service), func(t *testing.T) {
			service := want.file.Services().ByName(want.service)
			methodCount := len(want.methods)
			if want.service == "MfaService" {
				methodCount += 7 // New session policies pinned separately by TestV080MethodAuthorization.
			}
			if want.file.Services().Len() != 1 || service == nil || service.Methods().Len() != methodCount {
				t.Fatal("unreviewed IAM RPC surface")
			}
			for _, name := range want.methods {
				t.Run(string(name), func(t *testing.T) {
					method := service.Methods().ByName(name)
					if method == nil || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
						t.Fatal("missing IAM method or policy")
					}
					policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
					if len(policy.GetRules()) != 1 {
						t.Fatalf("unexpected rules: %v", policy)
					}
					rule := policy.GetRules()[0]
					if !slices.Equal(rule.GetCallers(), []string{"backend"}) || rule.GetAssertion() != authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED || len(rule.GetPermissions()) != 0 || rule.GetRecentAuthMaxAgeSeconds() != 0 {
						t.Fatalf("unexpected IAM policy: %v", rule)
					}
					if method.IsStreamingClient() || method.IsStreamingServer() {
						t.Fatal("IAM methods must remain unary")
					}
				})
			}
		})
	}
}

// Preserve every published credential branch; Telegram settings must never
// start accepting EPay or workload credentials. Presence distinguishes update
// ID zero from a missing replay identity.
func TestIAMTelegramCredentialBoundary(t *testing.T) {
	authorize := (&iamv1.AuthorizeRequest{}).ProtoReflect().Descriptor()
	credential := authorize.Oneofs().ByName("credential")
	branches := map[protoreflect.Name]protoreflect.FieldNumber{
		"bearer": 2, "epay_v1_signature": 3, "epay_v1_key": 4, "epay_v2_signature": 5, "workload_identity": 6, "telegram": 11,
	}
	if credential == nil || credential.Fields().Len() != len(branches) {
		t.Fatal("unreviewed Authorize credential set")
	}
	for name, number := range branches {
		field := credential.Fields().ByName(name)
		if field == nil || field.Number() != number || field.Kind() != protoreflect.MessageKind {
			t.Fatalf("published credential branch changed: %s", name)
		}
	}
	totp := authorize.Fields().ByName("totp_code")
	if totp == nil || totp.Number() != 12 || totp.Kind() != protoreflect.StringKind || totp.ContainingOneof() != nil {
		t.Fatal("TOTP must accompany, not replace, the Telegram credential")
	}
	human := (&iamv1.UserAuthentication{}).ProtoReflect().Descriptor().Oneofs().ByName("credential")
	if human == nil || human.Fields().Len() != 2 || human.Fields().ByName("bearer") == nil || human.Fields().ByName("telegram") == nil {
		t.Fatal("security settings must accept only human credentials")
	}
	update := (&iamv1.TelegramCredential{}).ProtoReflect().Descriptor().Fields().ByName("update_id")
	if update == nil || update.Number() != 4 || update.Kind() != protoreflect.Int64Kind || !update.HasPresence() {
		t.Fatal("update ID must distinguish valid zero from absence")
	}
	query := (&iamv1.GetGroupBindingResponse{}).ProtoReflect().Descriptor()
	if query.Fields().Len() != 2 || query.Fields().ByName("bound") == nil || query.Fields().ByName("is_owner") == nil {
		t.Fatal("group query must not disclose another owner's identity")
	}
	for _, want := range []struct {
		enum  protoreflect.EnumDescriptor
		names []protoreflect.Name
	}{
		{iamv1.TelegramChatType(0).Descriptor(), []protoreflect.Name{"TELEGRAM_CHAT_TYPE_UNSPECIFIED", "TELEGRAM_CHAT_TYPE_PRIVATE", "TELEGRAM_CHAT_TYPE_GROUP", "TELEGRAM_CHAT_TYPE_SUPERGROUP"}},
		{iamv1.TotpResetReason(0).Descriptor(), []protoreflect.Name{"TOTP_RESET_REASON_UNSPECIFIED", "TOTP_RESET_REASON_RECOVERY_CODE", "TOTP_RESET_REASON_PLATFORM_RESET"}},
	} {
		if want.enum.Values().Len() != len(want.names) {
			t.Fatalf("unreviewed %s value", want.enum.FullName())
		}
		for i, name := range want.names {
			value := want.enum.Values().Get(i)
			if value.Name() != name || value.Number() != protoreflect.EnumNumber(i) {
				t.Fatalf("unexpected %s value: %v", want.enum.FullName(), value)
			}
		}
	}
}
