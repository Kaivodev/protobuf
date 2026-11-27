package contracttest

import (
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	commonv1 "github.com/Kaivodev/protobuf/gen/go/aipay/common/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestV0250LedgerApprovalFields(t *testing.T) {
	const (
		msg             = protoreflect.MessageKind
		str             = protoreflect.StringKind
		data            = protoreflect.BytesKind
		u32             = protoreflect.Uint32Kind
		u64             = protoreflect.Uint64Kind
		i64             = protoreflect.Int64Kind
		ledger          = "aipay.ledger.v1."
		timestamp       = "google.protobuf.Timestamp"
		approval        = "aipay.approval.v1.Approval"
		approvalRequest = "aipay.approval.v1.ApprovalRequest"
	)
	type fields []struct {
		number      protoreflect.FieldNumber
		name        protoreflect.Name
		kind        protoreflect.Kind
		messageType protoreflect.FullName
		repeated    bool
	}
	for _, tc := range []struct {
		message proto.Message
		fields  fields
	}{
		{&ledgerv1.AdjustmentEvidenceUpload{}, fields{
			{1, "file_name", str, "", false},
			{2, "content_type", str, "", false},
			{3, "data", data, "", false},
		}},
		{&ledgerv1.AdjustmentEvidenceFile{}, fields{
			{1, "file_no", u32, "", false},
			{2, "file_name", str, "", false},
			{3, "content_type", str, "", false},
			{4, "size_bytes", u32, "", false},
			{5, "sha256", data, "", false},
		}},
		{&ledgerv1.SubmitAdjustmentRequest{}, fields{
			{1, "idempotency_key", str, "", false},
			{2, "reason", str, "", false},
			{3, "evidence_sha256", data, "", false},
			{4, "corrects_seq", u64, "", false},
			{5, "lines", msg, ledger + "AdjustmentLine", true},
			{6, "evidence_files", msg, ledger + "AdjustmentEvidenceUpload", true},
		}},
		{&ledgerv1.SubmitAdjustmentResponse{}, fields{
			{1, "approval_request", msg, approvalRequest, false},
			{2, "evidence_files", msg, ledger + "AdjustmentEvidenceFile", true},
		}},
		{&ledgerv1.AdjustmentMaterial{}, fields{
			{1, "reason", str, "", false},
			{2, "evidence_sha256", data, "", false},
			{3, "corrects_seq", u64, "", false},
			{4, "lines", msg, ledger + "AdjustmentLine", true},
			{5, "evidence_files", msg, ledger + "AdjustmentEvidenceFile", true},
		}},
		{&ledgerv1.CarryoverMaterial{}, fields{
			{1, "period_id", str, "", false},
			{2, "account_code", str, "", false},
			{3, "units", i64, "", false},
			{4, "reason", str, "", false},
		}},
		{&ledgerv1.UsdRatePolicyMaterial{}, fields{
			{1, "version", u64, "", false},
			{2, "expected_version", u64, "", false},
			{3, "threshold_ppm", u32, "", false},
			{4, "ceiling_ppm", u32, "", false},
			{5, "reason", str, "", false},
		}},
		{&ledgerv1.LedgerApprovalMaterial{}, fields{
			{1, "approval_request_id", str, "", false},
			{2, "kind", str, "", false},
			{3, "requested_by", str, "", false},
			{4, "requested_at", msg, timestamp, false},
			{5, "expires_at", msg, timestamp, false},
			{6, "adjustment", msg, ledger + "AdjustmentMaterial", false},
			{7, "read_only_exit", msg, ledger + "ReadOnlyExitSnapshot", false},
			{8, "carryover", msg, ledger + "CarryoverMaterial", false},
			{9, "usd_rate_policy", msg, ledger + "UsdRatePolicyMaterial", false},
		}},
		{&ledgerv1.LedgerAccountView{}, fields{
			{1, "account_code", str, "", false},
			{2, "account_type", str, "", false},
			{3, "currency", str, "", false},
			{4, "chain", str, "", false},
			{5, "owner_kind", str, "", false},
			{6, "owner_id", str, "", false},
			{7, "normal_side", protoreflect.EnumKind, "", false},
			{8, "non_negative", protoreflect.BoolKind, "", false},
			{9, "balance_units", i64, "", false},
			{10, "opens_on_execution", protoreflect.BoolKind, "", false},
		}},
		{&ledgerv1.AdjustmentReview{}, fields{
			{1, "accounts", msg, ledger + "LedgerAccountView", true},
			{2, "corrected_transaction", msg, ledger + "JournalTransaction", false},
			{3, "posted_transaction", msg, ledger + "JournalTransaction", false},
		}},
		{&ledgerv1.LedgerApproval{}, fields{
			{1, "approval_request", msg, approvalRequest, false},
			{2, "material", msg, ledger + "LedgerApprovalMaterial", false},
			{3, "approvals", msg, approval, true},
			{4, "approved_at", msg, timestamp, false},
			{5, "executed_at", msg, timestamp, false},
			{6, "adjustment_review", msg, ledger + "AdjustmentReview", false},
		}},
		{&ledgerv1.GetAdjustmentRequest{}, fields{
			{1, "approval_request_id", str, "", false},
		}},
		{&ledgerv1.GetAdjustmentResponse{}, fields{
			{1, "approval", msg, ledger + "LedgerApproval", false},
			{2, "as_of", msg, timestamp, false},
		}},
		{&ledgerv1.ListAdjustmentsRequest{}, fields{
			{1, "status", protoreflect.EnumKind, "", false},
			{2, "limit", u32, "", false},
			{3, "page_token", str, "", false},
		}},
		{&ledgerv1.ListAdjustmentsResponse{}, fields{
			{1, "adjustments", msg, ledger + "LedgerApproval", true},
			{2, "next_page_token", str, "", false},
			{3, "as_of", msg, timestamp, false},
		}},
		{&ledgerv1.GetAdjustmentEvidenceFileRequest{}, fields{
			{1, "approval_request_id", str, "", false},
			{2, "file_no", u32, "", false},
		}},
		{&ledgerv1.GetAdjustmentEvidenceFileResponse{}, fields{
			{1, "file", msg, ledger + "AdjustmentEvidenceFile", false},
			{2, "data", data, "", false},
		}},
		{&ledgerv1.ListApprovalsRequest{}, fields{
			{1, "kind", str, "", false},
			{2, "limit", u32, "", false},
			{3, "page_token", str, "", false},
		}},
		{&ledgerv1.ListApprovalsResponse{}, fields{
			{1, "approvals", msg, approvalRequest, true},
			{2, "next_page_token", str, "", false},
			{3, "as_of", msg, timestamp, false},
		}},
		{&ledgerv1.GetApprovalRequest{}, fields{
			{1, "approval_request_id", str, "", false},
		}},
		{&ledgerv1.GetApprovalResponse{}, fields{
			{1, "approval", msg, ledger + "LedgerApproval", false},
			{2, "as_of", msg, timestamp, false},
		}},
	} {
		desc := tc.message.ProtoReflect().Descriptor()
		t.Run(string(desc.Name()), func(t *testing.T) {
			if got := desc.Fields().Len(); got != len(tc.fields) {
				t.Fatalf("field count = %d, want %d", got, len(tc.fields))
			}
			for _, want := range tc.fields {
				field := requireField(t, tc.message, want.name, want.number, want.kind, want.messageType)
				if field.IsList() != want.repeated {
					t.Errorf("%s repeated = %t, want %t", field.FullName(), field.IsList(), want.repeated)
				}
				if desc.Name() == "LedgerApprovalMaterial" && want.number >= 6 {
					if group := field.ContainingOneof(); group == nil || group.Name() != "operation" {
						t.Errorf("%s must be a branch of operation", field.FullName())
					}
				} else if field.ContainingOneof() != nil {
					t.Errorf("unexpected oneof for %s", field.FullName())
				}
			}
		})
	}
}

func TestV0250LedgerApprovalEnumsAndBounds(t *testing.T) {
	for _, tc := range []struct {
		message proto.Message
		field   protoreflect.Name
		enum    protoreflect.FullName
	}{
		{&ledgerv1.LedgerAccountView{}, "normal_side", "aipay.ledger.v1.EntryDirection"},
		{&ledgerv1.ListAdjustmentsRequest{}, "status", "aipay.approval.v1.ApprovalStatus"},
	} {
		field := tc.message.ProtoReflect().Descriptor().Fields().ByName(tc.field)
		if field == nil || field.Enum() == nil || field.Enum().FullName() != tc.enum {
			t.Errorf("%T.%s must use %s", tc.message, tc.field, tc.enum)
		}
	}
	for _, tc := range []struct {
		message proto.Message
		field   protoreflect.Name
		limit   uint32
	}{
		{&ledgerv1.SubmitAdjustmentRequest{}, "evidence_files", 5},
		{&ledgerv1.ListAdjustmentsRequest{}, "limit", 100},
		{&ledgerv1.ListApprovalsRequest{}, "limit", 100},
	} {
		field := tc.message.ProtoReflect().Descriptor().Fields().ByName(tc.field)
		if field == nil || proto.GetExtension(field.Options(), commonv1.E_MaxItems) != tc.limit {
			t.Errorf("%T.%s must declare max_items = %d", tc.message, tc.field, tc.limit)
		}
	}
}

func TestV0250LedgerApprovalPolicies(t *testing.T) {
	writes := ledgerv1.File_aipay_ledger_v1_adjustment_service_proto
	reads := ledgerv1.File_aipay_ledger_v1_approval_service_proto
	protected := map[string]bool{}
	permittedMethods := map[protoreflect.FullName]string{}
	for _, tc := range []struct {
		file    protoreflect.FileDescriptor
		service protoreflect.Name
		methods map[protoreflect.Name]string
	}{
		{writes, "AdjustmentService", map[protoreflect.Name]string{
			"SubmitAdjustment":  "ledger.adjustment.submit",
			"ApproveAdjustment": "ledger.adjustment.approve",
		}},
		{reads, "AdjustmentQueryService", map[protoreflect.Name]string{
			"GetAdjustment":             "ledger.adjustment.read",
			"ListAdjustments":           "ledger.adjustment.read",
			"GetAdjustmentEvidenceFile": "ledger.adjustment.read",
		}},
		{reads, "LedgerApprovalService", map[protoreflect.Name]string{
			"ListApprovals": "ledger.approval.read",
			"GetApproval":   "ledger.approval.read",
		}},
	} {
		service := tc.file.Services().ByName(tc.service)
		if service == nil || service.Methods().Len() != len(tc.methods) {
			t.Fatalf("unexpected RPC set for %s", tc.service)
		}
		for name, permission := range tc.methods {
			method := service.Methods().ByName(name)
			if method == nil || method.IsStreamingClient() || method.IsStreamingServer() ||
				method.Input().FullName() != tc.file.Package().Append(name+"Request") ||
				method.Output().FullName() != tc.file.Package().Append(name+"Response") {
				t.Fatalf("unexpected signature for %s.%s", tc.service, name)
			}
			want := &authzv1.MethodPolicy{Rules: []*authzv1.CallerRule{{
				Callers: []string{"backend"}, Assertion: authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED,
				Permissions: []string{permission},
			}}}
			got := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
			if !proto.Equal(got, want) {
				t.Errorf("%s policy = %v, want %v", method.FullName(), got, want)
			}
			protected[permission] = true
			permittedMethods[method.FullName()] = permission
		}
	}
	if len(protected) != 4 {
		t.Fatal("adjustment submit, approve, read and approval read require distinct permissions")
	}
	protoregistry.GlobalFiles.RangeFilesByPackage(writes.Package(), func(file protoreflect.FileDescriptor) bool {
		for i := 0; i < file.Services().Len(); i++ {
			methods := file.Services().Get(i).Methods()
			for j := 0; j < methods.Len(); j++ {
				method := methods.Get(j)
				policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
				for _, rule := range policy.GetRules() {
					for _, permission := range rule.GetPermissions() {
						if protected[permission] && permittedMethods[method.FullName()] != permission {
							t.Errorf("%s reuses permission %q", method.FullName(), permission)
						}
					}
				}
			}
		}
		return true
	})
}
