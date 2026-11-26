package contracttest

import (
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	commonv1 "github.com/Kaivodev/protobuf/gen/go/aipay/common/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	reconciliationv1 "github.com/Kaivodev/protobuf/gen/go/aipay/reconciliation/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0200CoveragePolicies(t *testing.T) {
	admin := reconciliationv1.File_aipay_reconciliation_v1_reconciliation_admin_service_proto.Services().ByName("ReconciliationAdminService")
	journal := ledgerv1.File_aipay_ledger_v1_journal_service_proto.Services().ByName("JournalService")
	const (
		recent   = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT
		required = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED
		workload = authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED
	)
	for _, tc := range []struct {
		service    protoreflect.ServiceDescriptor
		method     protoreflect.Name
		caller     string
		assertion  authzv1.AssertionRequirement
		permission string
	}{
		{admin, "SubmitCoverageAttestation", "backend", recent, "reconciliation.coverage.submit"},
		{admin, "ApproveCoverageAttestation", "backend", recent, "reconciliation.coverage.approve"},
		{admin, "RevokeCoverageAttestation", "backend", recent, "reconciliation.coverage.revoke"},
		{admin, "GetCoverageAttestation", "backend", required, "reconciliation.coverage.read"},
		{admin, "ListCoverageAttestations", "backend", required, "reconciliation.coverage.read"},
		{admin, "GetCoverageStatus", "backend", required, "reconciliation.coverage.read"},
		{admin, "ListCoverageAddresses", "backend", required, "reconciliation.coverage.read"},
		{journal, "GetLiabilitySnapshot", "reconciliation", workload, ""},
	} {
		t.Run(string(tc.method), func(t *testing.T) {
			m := tc.service.Methods().ByName(tc.method)
			if m == nil || m.IsStreamingClient() || m.IsStreamingServer() || !proto.HasExtension(m.Options(), authzv1.E_Policy) {
				t.Fatal("expected unary RPC with explicit policy")
			}
			if m.Input().Name() != tc.method+"Request" || m.Output().Name() != tc.method+"Response" {
				t.Fatal("unexpected RPC input or output")
			}
			rule := &authzv1.CallerRule{Callers: []string{tc.caller}, Assertion: tc.assertion}
			if tc.permission != "" {
				rule.Permissions = []string{tc.permission}
			}
			if tc.assertion == recent {
				rule.RecentAuthMaxAgeSeconds = 600
			}
			got := proto.GetExtension(m.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
			if !proto.Equal(got, &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{rule}}) {
				t.Fatalf("unexpected caller rules: %v", got)
			}
		})
	}
}

func TestV0200CoverageMaterial(t *testing.T) {
	material := &reconciliationv1.ReconciliationApprovalMaterial{}
	for _, tc := range []struct {
		name    protoreflect.Name
		number  protoreflect.FieldNumber
		message protoreflect.FullName
	}{
		{"verification_account", 6, "aipay.reconciliation.v1.VerificationAccountChange"},
		{"plugin_version", 7, "aipay.reconciliation.v1.PluginVersionMaterial"},
		{"plugin_enable", 8, "aipay.reconciliation.v1.PluginEnableChange"},
		{"coverage_attestation", 9, "aipay.reconciliation.v1.CoverageAttestationMaterial"},
	} {
		f := requireField(t, material, tc.name, tc.number, protoreflect.MessageKind, tc.message)
		if f.ContainingOneof() == nil || f.ContainingOneof().Name() != "operation" {
			t.Errorf("%s must select the operation", f.FullName())
		}
	}
	for _, field := range []protoreflect.FieldDescriptor{
		requireField(t, &reconciliationv1.CoverageLedgerSnapshot{}, "checkpoint_id", 7, protoreflect.Uint64Kind, ""),
		requireField(t, &reconciliationv1.CoverageAttestationMaterial{}, "note", 15, protoreflect.StringKind, ""),
		requireField(t, &reconciliationv1.SubmitCoverageAttestationRequest{}, "note", 5, protoreflect.StringKind, ""),
	} {
		if !field.HasOptionalKeyword() {
			t.Errorf("%s requires explicit presence", field.FullName())
		}
	}
}

func TestV0200CoverageLimits(t *testing.T) {
	submit := &reconciliationv1.SubmitCoverageAttestationRequest{}
	material := &reconciliationv1.CoverageAttestationMaterial{}
	for _, tc := range []struct {
		message proto.Message
		field   protoreflect.Name
		number  protoreflect.FieldNumber
		limit   uint32
	}{
		{submit, "observations", 2, 200},
		{submit, "unrecorded_inflows", 3, 200},
		{submit, "window_outflows", 4, 200},
		{material, "observations", 5, 200},
		{material, "unrecorded_inflows", 6, 200},
		{material, "window_outflows", 7, 200},
		{&reconciliationv1.ListCoverageAttestationsRequest{}, "limit", 1, 100},
		{&reconciliationv1.ListCoverageAddressesRequest{}, "limit", 1, 500},
		{&ledgerv1.GetLiabilitySnapshotRequest{}, "limit", 3, 2000},
	} {
		f := tc.message.ProtoReflect().Descriptor().Fields().ByName(tc.field)
		if f == nil || f.Number() != tc.number || f.IsList() != (tc.field != "limit") {
			t.Fatalf("missing or changed field %T.%s", tc.message, tc.field)
		}
		if got := proto.GetExtension(f.Options(), commonv1.E_MaxItems); got != tc.limit {
			t.Errorf("%s max_items = %v, want %d", f.FullName(), got, tc.limit)
		}
	}
}
