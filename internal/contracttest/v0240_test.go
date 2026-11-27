package contracttest

import (
	"strings"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestV0240LedgerControlFields(t *testing.T) {
	const (
		msg = protoreflect.MessageKind
		str = protoreflect.StringKind

		timestamp       = "google.protobuf.Timestamp"
		approval        = "aipay.approval.v1.Approval"
		approvalRequest = "aipay.approval.v1.ApprovalRequest"
	)
	type field struct {
		number      protoreflect.FieldNumber
		name        protoreflect.Name
		kind        protoreflect.Kind
		messageType protoreflect.FullName
		repeated    bool
	}
	for _, tc := range []struct {
		message proto.Message
		fields  []field
	}{
		{&ledgerv1.PendingReadOnlyExit{}, []field{
			{1, "approval_request", msg, approvalRequest, false},
			{2, "snapshot", msg, "aipay.ledger.v1.ReadOnlyExitSnapshot", false},
			{3, "approvals", msg, approval, true},
		}},
		{&ledgerv1.GetReadOnlyStatusRequest{}, nil},
		{&ledgerv1.GetReadOnlyStatusResponse{}, []field{
			{1, "mode", protoreflect.EnumKind, "", false},
			{2, "read_only_reason", str, "", false},
			{3, "mode_since", msg, timestamp, false},
			{4, "state_version", protoreflect.Uint64Kind, "", false},
			{5, "last_exit_request_id", str, "", false},
			{6, "pending", msg, "aipay.ledger.v1.PendingReadOnlyExit", true},
			{7, "as_of", msg, timestamp, false},
		}},
		{&ledgerv1.SuspenseBalance{}, []field{
			{1, "account_code", str, "", false},
			{2, "account_type", str, "", false},
			{3, "units", protoreflect.Int64Kind, "", false},
			{4, "carryover_approval_request_id", str, "", false},
		}},
		{&ledgerv1.PeriodCarryover{}, []field{
			{1, "approval_request", msg, approvalRequest, false},
			{2, "account_code", str, "", false},
			{3, "units", protoreflect.Int64Kind, "", false},
			{4, "reason", str, "", false},
			{5, "approvals", msg, approval, true},
		}},
		{&ledgerv1.GetPeriodCarryoversRequest{}, []field{
			{1, "period_id", str, "", false},
		}},
		{&ledgerv1.GetPeriodCarryoversResponse{}, []field{
			{1, "period_id", str, "", false},
			{2, "status", protoreflect.EnumKind, "", false},
			{3, "starts_at", msg, timestamp, false},
			{4, "ends_at", msg, timestamp, false},
			{5, "ended", protoreflect.BoolKind, "", false},
			{6, "balances", msg, "aipay.ledger.v1.SuspenseBalance", true},
			{7, "requests", msg, "aipay.ledger.v1.PeriodCarryover", true},
			{8, "truncated", protoreflect.BoolKind, "", false},
			{9, "as_of", msg, timestamp, false},
		}},
	} {
		desc := tc.message.ProtoReflect().Descriptor()
		t.Run(string(desc.Name()), func(t *testing.T) {
			if got := desc.Fields().Len(); got != len(tc.fields) {
				t.Fatalf("field count = %d, want %d", got, len(tc.fields))
			}
			for _, want := range tc.fields {
				f := requireField(t, tc.message, want.name, want.number, want.kind, want.messageType)
				if f.IsList() != want.repeated || f.ContainingOneof() != nil {
					t.Errorf("unexpected cardinality or oneof for %s", f.FullName())
				}
			}
		})
	}
}

func TestV0240LedgerControlEnums(t *testing.T) {
	for _, tc := range []struct {
		field  protoreflect.FieldDescriptor
		name   protoreflect.FullName
		values []protoreflect.Name
	}{
		{
			requireField(t, &ledgerv1.GetReadOnlyStatusResponse{}, "mode", 1, protoreflect.EnumKind, ""),
			"aipay.ledger.v1.LedgerServiceMode",
			[]protoreflect.Name{"LEDGER_SERVICE_MODE_UNSPECIFIED", "LEDGER_SERVICE_MODE_NORMAL", "LEDGER_SERVICE_MODE_READ_ONLY"},
		},
		{
			requireField(t, &ledgerv1.GetPeriodCarryoversResponse{}, "status", 2, protoreflect.EnumKind, ""),
			"aipay.ledger.v1.PeriodStatus",
			[]protoreflect.Name{"PERIOD_STATUS_UNSPECIFIED", "PERIOD_STATUS_OPEN", "PERIOD_STATUS_CLOSING", "PERIOD_STATUS_CLOSED"},
		},
	} {
		t.Run(string(tc.name), func(t *testing.T) {
			enum := tc.field.Enum()
			if enum.FullName() != tc.name {
				t.Fatalf("%s uses enum %s, want %s", tc.field.FullName(), enum.FullName(), tc.name)
			}
			if got := enum.Values().Len(); got != len(tc.values) {
				t.Fatalf("enum value count = %d, want %d", got, len(tc.values))
			}
			for number, name := range tc.values {
				value := enum.Values().ByNumber(protoreflect.EnumNumber(number))
				if value == nil || value.Name() != name {
					t.Errorf("enum value %d must be %s", number, name)
				}
			}
		})
	}
}

func TestV0240LedgerControlPermissions(t *testing.T) {
	service := ledgerv1.File_aipay_ledger_v1_control_service_proto.Services().ByName("LedgerControlService")
	seen := make(map[string]protoreflect.Name)
	for i := 0; i < service.Methods().Len(); i++ {
		method := service.Methods().Get(i)
		name := method.Name()
		t.Run(string(name), func(t *testing.T) {
			pkg := service.ParentFile().Package()
			if method.IsStreamingClient() || method.IsStreamingServer() || method.Input().FullName() != pkg.Append(name+"Request") || method.Output().FullName() != pkg.Append(name+"Response") {
				t.Fatal("unexpected RPC signature")
			}
			policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
			if len(policy.GetRules()) != 1 || len(policy.GetRules()[0].GetPermissions()) != 1 {
				t.Fatal("expected one caller rule with one permission")
			}
			permission := policy.GetRules()[0].GetPermissions()[0]
			if other, ok := seen[permission]; ok {
				t.Errorf("%s and %s share permission %q", other, name, permission)
			}
			seen[permission] = name
			if strings.HasPrefix(permission, "ledger.control.") {
				t.Errorf("permission %q uses the retired prefix", permission)
			}
			var suffix string
			switch name {
			case "GetReadOnlyStatus", "GetPeriodCarryovers", "GetUsdRatePolicy":
				suffix = ".read"
			case "RequestReadOnlyExit", "RecordCarryover", "RequestUsdRatePolicy":
				suffix = ".submit"
			case "ApproveReadOnlyExit", "ApproveCarryover", "ApproveUsdRatePolicy":
				suffix = ".approve"
			default:
				t.Fatalf("unreviewed ledger control method %s", name)
			}
			if !strings.HasSuffix(permission, suffix) {
				t.Errorf("permission %q must end in %q", permission, suffix)
			}
		})
	}
}
