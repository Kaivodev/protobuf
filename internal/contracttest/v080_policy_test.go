package contracttest

import (
	"slices"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	iamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/iam/v1"
	reconciliationv1 "github.com/Kaivodev/protobuf/gen/go/aipay/reconciliation/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Pin every new RPC and the complete MFA surface. Service-only bootstrap must
// not become a general bypass; application writes must retain IAM-verified MFA.
// Scope/entry/subject conditions are business checks documented in contracts.md;
// CallerRule cannot encode them, and this test does not claim runtime coverage.
func TestV080MethodAuthorization(t *testing.T) {
	const (
		N = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED
		R = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED
		M = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT
	)
	type expected struct {
		assertion  authzv1.AssertionRequirement
		permission string
	}
	for _, group := range []struct {
		file    protoreflect.FileDescriptor
		service protoreflect.Name
		methods map[protoreflect.Name]expected
	}{
		{iamv1.File_aipay_iam_v1_account_service_proto, "AccountService", map[protoreflect.Name]expected{
			"Login":                 {N, ""},
			"VerifyChallenge":       {N, ""},
			"Logout":                {R, "iam.session.manage"},
			"BeginReauthentication": {N, ""},
			"ForgotPassword":        {N, ""},
			"ResetPassword":         {N, ""},
			"ChangePassword":        {M, "iam.password.change"},
			"BeginEmailChange":      {M, "iam.email.change"},
			"ConfirmEmailChange":    {N, ""},
			"GetProfile":            {R, "iam.profile.read"},
			"UpdateProfile":         {R, "iam.profile.manage"},
		}},
		{iamv1.File_aipay_iam_v1_session_service_proto, "SessionService", map[protoreflect.Name]expected{
			"ListSessions":      {R, "iam.session.read"},
			"RevokeSession":     {R, "iam.session.manage"},
			"RevokeAllSessions": {R, "iam.session.manage"},
		}},
		{iamv1.File_aipay_iam_v1_invitation_service_proto, "InvitationService", map[protoreflect.Name]expected{
			"CreateInvitation":  {M, "iam.invitation.manage"},
			"ApproveInvitation": {M, "iam.invitation.approve"},
			"InspectInvitation": {N, ""},
			"AcceptInvitation":  {N, ""},
			"RevokeInvitation":  {M, "iam.invitation.manage"},
			"ListInvitations":   {R, "iam.invitation.read"},
		}},
		{iamv1.File_aipay_iam_v1_mfa_service_proto, "MfaService", map[protoreflect.Name]expected{
			"IssueTotpEnrollmentToken":     {N, ""},
			"GetTotpEnrollment":            {N, ""},
			"CompleteTotpEnrollment":       {N, ""},
			"UseRecoveryCode":              {N, ""},
			"ResetTotp":                    {N, ""},
			"BeginSessionTotpEnrollment":   {R, "iam.mfa.manage"},
			"ConfirmSessionTotpEnrollment": {R, "iam.mfa.manage"},
			"BeginWebAuthnRegistration":    {R, "iam.mfa.manage"},
			"CompleteWebAuthnRegistration": {R, "iam.mfa.manage"},
			"RemoveWebAuthnCredential":     {M, "iam.mfa.manage"},
			"RotateRecoveryCodes":          {M, "iam.mfa.manage"},
			"ListMfaMethods":               {R, "iam.mfa.read"},
		}},
		{iamv1.File_aipay_iam_v1_business_service_proto, "BusinessService", map[protoreflect.Name]expected{
			"CreateBusiness": {M, "iam.business.manage"},
			"ListBusinesses": {R, "iam.business.read"},
			"GetBusiness":    {R, "iam.business.read"},
			"UpdateBusiness": {M, "iam.business.manage"},
		}},
		{iamv1.File_aipay_iam_v1_application_service_proto, "ApplicationService", map[protoreflect.Name]expected{
			"CreateApplication":          {M, "iam.application.manage"},
			"ListApplications":           {R, "iam.application.read"},
			"GetApplication":             {R, "iam.application.read"},
			"SetApplicationEnabled":      {M, "iam.application.manage"},
			"GenerateEpayV1Key":          {M, "iam.application.manage"},
			"RotateEpayV1Key":            {M, "iam.application.manage"},
			"UploadEpayV2PublicKey":      {M, "iam.application.manage"},
			"GetPlatformEpayV2PublicKey": {R, "iam.application.read"},
		}},
		{reconciliationv1.File_aipay_reconciliation_v1_reconciliation_admin_service_proto, "ReconciliationAdminService", map[protoreflect.Name]expected{
			"UploadPlugin":                 {R, "reconciliation.plugin.upload"},
			"ApprovePlugin":                {R, "reconciliation.plugin.approve"},
			"StoreVerificationCredentials": {R, "reconciliation.account.manage"},
			"SubmitVerificationAccount":    {R, "reconciliation.account.manage"},
			"ApproveVerificationAccount":   {R, "reconciliation.account.approve"},
			"RequestPluginEnable":          {R, "reconciliation.plugin.enable"},
			"ApprovePluginEnable":          {R, "reconciliation.plugin.approve"},
			"DisableVerificationAccount":   {R, "reconciliation.account.disable"},
			"ListVerificationAccounts":     {R, "reconciliation.account.read"},
			"GetVerificationAccount":       {R, "reconciliation.account.read"},
			"ListPlugins":                  {R, "reconciliation.plugin.read"},
			"GetPlugin":                    {R, "reconciliation.plugin.read"},
			"GetApproval":                  {R, "reconciliation.approval.read"},
		}},
	} {
		t.Run(string(group.service), func(t *testing.T) {
			service := group.file.Services().ByName(group.service)
			if group.file.Services().Len() != 1 || service == nil || service.Methods().Len() != len(group.methods) {
				t.Fatal("unreviewed RPC surface")
			}
			for name, want := range group.methods {
				t.Run(string(name), func(t *testing.T) {
					method := service.Methods().ByName(name)
					if method == nil || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
						t.Fatal("missing RPC or method policy")
					}
					policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
					if len(policy.GetRules()) != 1 {
						t.Fatalf("unexpected caller rules: %v", policy)
					}
					rule := policy.GetRules()[0]
					var permissions []string
					if want.permission != "" {
						permissions = []string{want.permission}
					}
					var maxAge uint32
					if want.assertion == M {
						maxAge = 600
					}
					if !slices.Equal(rule.GetCallers(), []string{"backend"}) || rule.GetAssertion() != want.assertion || !slices.Equal(rule.GetPermissions(), permissions) || rule.GetRecentAuthMaxAgeSeconds() != maxAge {
						t.Fatalf("unexpected policy: %v", rule)
					}
					if method.IsStreamingClient() || method.IsStreamingServer() {
						t.Fatal("must remain unary")
					}
				})
			}
		})
	}
}
