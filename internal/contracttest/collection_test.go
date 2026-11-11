package contracttest

import (
	"slices"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	collectionv1 "github.com/Kaivodev/protobuf/gen/go/aipay/collection/v1"
	iamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/iam/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// These tests pin the public descriptor, not runtime authorization or execution.
func TestCollectionContract(t *testing.T) {
	if iamv1.PlatformRole_PLATFORM_ROLE_COLLECTION_OPERATOR.Number() != 9 {
		t.Fatal("collection operator must have its own stable invitation role number")
	}
	file := collectionv1.File_aipay_collection_v1_collection_service_proto
	service := file.Services().ByName("CollectionService")
	if file.Services().Len() != 1 || service == nil || service.Methods().Len() != 3 {
		t.Fatal("collection must expose only create/get/cancel")
	}
	for _, want := range []struct {
		name       protoreflect.Name
		permission string
		assertion  authzv1.AssertionRequirement
		maxAge     uint32
	}{
		{"CreateSweepJob", "collection.sweep.create", authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT, 600},
		{"GetSweepJob", "collection.sweep.read", authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED, 0},
		{"CancelSweepJob", "collection.sweep.cancel", authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_MFA_RECENT, 600},
	} {
		t.Run(string(want.name), func(t *testing.T) {
			method := service.Methods().ByName(want.name)
			if method == nil || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
				t.Fatal("missing method policy")
			}
			policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
			if len(policy.GetRules()) != 1 {
				t.Fatal("unreviewed caller rule")
			}
			rule := policy.GetRules()[0]
			if !slices.Equal(rule.GetCallers(), []string{"backend"}) || !slices.Equal(rule.GetPermissions(), []string{want.permission}) || rule.GetAssertion() != want.assertion || rule.GetRecentAuthMaxAgeSeconds() != want.maxAge {
				t.Fatalf("unexpected collection policy: %v", rule)
			}
			if method.IsStreamingClient() || method.IsStreamingServer() {
				t.Fatal("collection methods are unary")
			}
		})
	}

	// Canonical encoding depends on these exact string fields and oneof numbers.
	// Extra target/nonce/gas/payload inputs would widen the approved surface.
	for _, want := range []struct {
		message proto.Message
		fields  []protoreflect.Name
	}{
		{&collectionv1.CreateSweepJobRequest{}, []protoreflect.Name{"idempotency_key", "account_id", "address_id", "amount_atomic", "reason"}},
		{&collectionv1.GetSweepJobRequest{}, []protoreflect.Name{"account_id", "sweep_id", "create_idempotency_key"}},
		{&collectionv1.CancelSweepJobRequest{}, []protoreflect.Name{"idempotency_key", "account_id", "sweep_id", "reason"}},
	} {
		if want.message.ProtoReflect().Descriptor().Fields().Len() != len(want.fields) {
			t.Fatal("unreviewed command field")
		}
		for i, name := range want.fields {
			requireField(t, want.message, name, protoreflect.FieldNumber(i+1), protoreflect.StringKind, "")
		}
	}
	selector := (&collectionv1.GetSweepJobRequest{}).ProtoReflect().Descriptor().Oneofs().ByName("selector")
	if selector == nil || selector.Fields().Len() != 2 || selector.Fields().Get(0).Number() != 2 || selector.Fields().Get(1).Number() != 3 {
		t.Fatal("lookup must distinguish sweep ID from original creation key")
	}
}

func TestCollectionEvidencePresence(t *testing.T) {
	job := &collectionv1.SweepJob{}
	if job.ProtoReflect().Descriptor().Fields().Len() != 17 {
		t.Fatal("unreviewed task projection")
	}
	for _, want := range []struct {
		name protoreflect.Name
		num  protoreflect.FieldNumber
	}{
		{"user_op_hash", 14}, {"transaction_hash", 15}, {"user_op_transaction_hash", 16},
	} {
		requireField(t, job, want.name, want.num, protoreflect.BytesKind, "")
	}
	for _, name := range []protoreflect.Name{"execution_phase", "blocker"} {
		field := job.ProtoReflect().Descriptor().Fields().ByName(name)
		if field == nil || !field.HasOptionalKeyword() || field.Kind() != protoreflect.EnumKind {
			t.Fatal("unknown execution evidence must remain absent")
		}
	}
	gas := &collectionv1.SweepGas{}
	for i, name := range []protoreflect.Name{"user_op_gas_used", "user_op_cost_atomic", "batch_transaction_gas_used", "batch_transaction_cost_atomic"} {
		if !requireField(t, gas, name, protoreflect.FieldNumber(i+1), protoreflect.StringKind, "").HasOptionalKeyword() {
			t.Fatal("unknown gas must differ from observed zero")
		}
	}
	// Consumers can preserve observed zero through the generated wire contract.
	gas.UserOpCostAtomic = proto.String("0")
	raw, err := proto.Marshal(gas)
	if err != nil {
		t.Fatal(err)
	}
	var decoded collectionv1.SweepGas
	if err := proto.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.UserOpCostAtomic == nil || *decoded.UserOpCostAtomic != "0" || decoded.BatchTransactionCostAtomic != nil {
		t.Fatal("gas round trip conflates unknown and zero")
	}
}

func TestDepositAddressContract(t *testing.T) {
	service := collectionv1.File_aipay_collection_v1_deposit_address_service_proto.Services().ByName("DepositAddressService")
	if service == nil || service.Methods().Len() != 2 {
		t.Fatal("address service must expose only ensure and lookup")
	}
	for _, want := range []struct {
		name       protoreflect.Name
		permission string
	}{
		{"GetOrCreateDepositAddress", "collection.address.ensure"},
		{"GetAddressAllocation", "collection.address.read"},
	} {
		method := service.Methods().ByName(want.name)
		if method == nil || !proto.HasExtension(method.Options(), authzv1.E_Policy) {
			t.Fatal("missing address method policy")
		}
		policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
		if len(policy.GetRules()) != 1 {
			t.Fatal("unreviewed address caller rule")
		}
		rule := policy.GetRules()[0]
		if !slices.Equal(rule.GetCallers(), []string{"backend"}) || !slices.Equal(rule.GetPermissions(), []string{want.permission}) || rule.GetAssertion() != authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_REQUIRED || rule.GetRecentAuthMaxAgeSeconds() != 0 {
			t.Fatal("address authority differs from account proof policy", rule)
		}
	}
	request := &collectionv1.GetOrCreateDepositAddressRequest{}
	if request.ProtoReflect().Descriptor().Fields().Len() != 3 {
		t.Fatal("address callers must not select account, target or signing parameters")
	}
	for i, name := range []protoreflect.Name{"request_id", "network_id", "asset_id"} {
		requireField(t, request, name, protoreflect.FieldNumber(i+1), protoreflect.StringKind, "")
	}
	selector := (&collectionv1.GetAddressAllocationRequest{}).ProtoReflect().Descriptor().Oneofs().ByName("selector")
	if selector == nil || selector.Fields().Len() != 2 || selector.Fields().Get(0).Name() != "allocation_id" || selector.Fields().Get(1).Name() != "request_id" {
		t.Fatal("lookup must distinguish allocation from original request")
	}
}
