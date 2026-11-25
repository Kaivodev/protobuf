package contracttest

import (
	"slices"
	"testing"

	authzv1 "github.com/Kaivodev/protobuf/gen/go/aipay/authz/v1"
	upstreamv1 "github.com/Kaivodev/protobuf/gen/go/aipay/upstream/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const upstreamPackage protoreflect.FullName = "aipay.upstream.v1"

var upstreamFile = upstreamv1.File_aipay_upstream_v1_upstream_service_proto

func TestUpstreamFile(t *testing.T) {
	if got := upstreamFile.Package(); got != upstreamPackage {
		t.Errorf("package = %s", got)
	}
	if got := upstreamFile.Options().(*descriptorpb.FileOptions).GetGoPackage(); got != "github.com/Kaivodev/protobuf/gen/go/aipay/upstream/v1;upstreamv1" {
		t.Errorf("go_package = %s", got)
	}
}

func TestUpstreamMethods(t *testing.T) {
	trade := []string{"payment"}
	both := []string{"payment", "reconciliation"}
	services := map[protoreflect.Name]map[protoreflect.Name][]string{
		"UpstreamService": {
			"CreatePayment": trade, "QueryPayment": both, "ClosePayment": trade,
			"Refund": trade, "QueryRefund": both, "RouteCallback": trade, "VerifyCallback": trade,
		},
		"UpstreamAdminService": {
			"ReplayPackage": both, "RegisterPackage": both, "RevokePackage": both,
			"StoreCredentials": both, "GetExchange": both,
		},
	}
	if upstreamFile.Services().Len() != len(services) {
		t.Fatalf("unexpected service count %d", upstreamFile.Services().Len())
	}
	for serviceName, methods := range services {
		service := upstreamFile.Services().ByName(serviceName)
		if service == nil || service.Methods().Len() != len(methods) {
			t.Fatalf("%s: unexpected RPC set", serviceName)
		}
		for name, callers := range methods {
			fullName := upstreamPackage.Append(serviceName).Append(name)
			t.Run(string(fullName), func(t *testing.T) {
				method := service.Methods().ByName(name)
				if method == nil || method.FullName() != fullName {
					t.Fatalf("missing method %s", fullName)
				}
				if method.Input().FullName() != upstreamPackage.Append(name+"Request") ||
					method.Output().FullName() != upstreamPackage.Append(name+"Response") ||
					method.IsStreamingClient() || method.IsStreamingServer() {
					t.Fatalf("unexpected signature: %v", method)
				}
				if !proto.HasExtension(method.Options(), authzv1.E_Policy) {
					t.Fatal("missing policy")
				}
				policy := proto.GetExtension(method.Options(), authzv1.E_Policy).(*authzv1.MethodPolicy)
				if len(policy.GetRules()) != 1 {
					t.Fatalf("unexpected policy: %v", policy)
				}
				rule := policy.GetRules()[0]
				got := slices.Clone(rule.GetCallers())
				slices.Sort(got)
				if !slices.Equal(got, callers) || rule.GetAssertion() != authzv1.AssertionRequirement_ASSERTION_REQUIREMENT_NOT_REQUIRED ||
					len(rule.GetPermissions()) != 0 || rule.GetRecentAuthMaxAgeSeconds() != 0 {
					t.Fatalf("unexpected authorization: %v", rule)
				}
			})
		}
	}
}

func TestUpstreamEnums(t *testing.T) {
	enums := map[protoreflect.Name][]protoreflect.Name{
		"ExecutionCategory": {"EXECUTION_CATEGORY_UNSPECIFIED", "EXECUTION_CATEGORY_SUCCEEDED", "EXECUTION_CATEGORY_REJECTED", "EXECUTION_CATEGORY_UNKNOWN", "EXECUTION_CATEGORY_NOT_SENT"},
		"PaymentStatus":     {"PAYMENT_STATUS_UNSPECIFIED", "PAYMENT_STATUS_PENDING", "PAYMENT_STATUS_PAID", "PAYMENT_STATUS_CLOSED", "PAYMENT_STATUS_FAILED", "PAYMENT_STATUS_UNKNOWN"},
		"RefundStatus":      {"REFUND_STATUS_UNSPECIFIED", "REFUND_STATUS_PENDING", "REFUND_STATUS_SUCCEEDED", "REFUND_STATUS_FAILED", "REFUND_STATUS_UNKNOWN"},
	}
	if upstreamFile.Enums().Len() != len(enums) {
		t.Fatalf("unexpected enum count %d", upstreamFile.Enums().Len())
	}
	for name, values := range enums {
		enum := upstreamFile.Enums().ByName(name)
		if enum == nil || enum.Values().Len() != len(values) {
			t.Fatalf("%s: unexpected enum values", name)
		}
		for number, valueName := range values {
			value := enum.Values().ByName(valueName)
			if value == nil || value.Number() != protoreflect.EnumNumber(number) {
				t.Errorf("%s: expected %s = %d", name, valueName, number)
			}
		}
	}
}

func TestUpstreamMessages(t *testing.T) {
	type fieldSpec struct {
		number protoreflect.FieldNumber
		name   protoreflect.Name
		// Scalar kind or fully qualified message/enum name, with the proto field label when present.
		fieldType string
	}
	const pkg = string(upstreamPackage) + "."
	messages := map[protoreflect.Name][]fieldSpec{
		"OperationContext": {
			{1, "idempotency_key", "string"},
			{2, "package_sha256", "bytes"},
			{3, "channel_account_id", "string"},
			{4, "verification_account_id", "optional string"},
			{5, "control_version", "uint64"},
			{6, "config_json", "bytes"},
			{7, "credentials_ref", "string"},
			{8, "credentials_sha256", "bytes"},
		},
		"ExecutionResult": {
			{1, "category", pkg + "ExecutionCategory"},
			{2, "error_code", "string"},
			{3, "exchange_id", "string"},
			{4, "evidence_sha256", "bytes"},
			{5, "package_sha256", "bytes"},
			{6, "requests", "uint32"},
			{7, "responses", "uint32"},
			{8, "steps", "uint32"},
			{9, "upstream_code", "optional string"},
		},
		"Money": {
			{1, "amount", "string"},
			{2, "currency", "string"},
		},
		"HttpHeader": {
			{1, "name", "string"},
			{2, "values", "repeated string"},
		},
		"HttpResponse": {
			{1, "status", "uint32"},
			{2, "headers", "repeated " + pkg + "HttpHeader"},
			{3, "body", "bytes"},
		},
		"CreatePaymentInput": {
			{1, "order_id", "string"},
			{2, "money", pkg + "Money"},
			{3, "method", "string"},
			{4, "callback_url", "string"},
			{5, "bank_mode", "optional string"},
			{6, "instruction", "optional string"},
		},
		"PaymentInput": {
			{1, "order_id", "string"},
			{2, "money", pkg + "Money"},
		},
		"RefundInput": {
			{1, "order_id", "string"},
			{2, "refund_id", "string"},
			{3, "money", pkg + "Money"},
		},
		"CallbackPayload": {
			{1, "method", "string"},
			{2, "raw_query", "bytes"},
			{3, "headers", "repeated " + pkg + "HttpHeader"},
			{4, "body", "bytes"},
			{5, "source_ip", "optional string"},
		},
		"VerifyCallbackInput": {
			{1, "order_id", "string"},
			{2, "money", pkg + "Money"},
			{3, "raw", pkg + "CallbackPayload"},
		},
		"PaymentOutput": {
			{1, "order_id", "string"},
			{2, "upstream_id", "string"},
			{3, "money", pkg + "Money"},
			{4, "status", pkg + "PaymentStatus"},
			{5, "url", "optional string"},
			{6, "qrcode", "optional string"},
			{7, "paid_at", "google.protobuf.Timestamp"},
			{8, "expires_at", "google.protobuf.Timestamp"},
		},
		"RefundOutput": {
			{1, "order_id", "string"},
			{2, "refund_id", "string"},
			{3, "upstream_refund_id", "string"},
			{4, "money", pkg + "Money"},
			{5, "status", pkg + "RefundStatus"},
		},
		"RouteCallbackOutput": {
			{1, "order_id", "string"},
		},
		"CallbackOutput": {
			{1, "event_id", "string"},
			{2, "order_id", "string"},
			{3, "upstream_id", "string"},
			{4, "money", pkg + "Money"},
			{5, "status", pkg + "PaymentStatus"},
			{6, "ack", pkg + "HttpResponse"},
		},
		"CreatePaymentRequest": {
			{1, "context", pkg + "OperationContext"},
			{2, "input", pkg + "CreatePaymentInput"},
		},
		"CreatePaymentResponse": {
			{1, "execution", pkg + "ExecutionResult"},
			{2, "output", pkg + "PaymentOutput"},
		},
		"QueryPaymentRequest": {
			{1, "context", pkg + "OperationContext"},
			{2, "input", pkg + "PaymentInput"},
		},
		"QueryPaymentResponse": {
			{1, "execution", pkg + "ExecutionResult"},
			{2, "output", pkg + "PaymentOutput"},
		},
		"ClosePaymentRequest": {
			{1, "context", pkg + "OperationContext"},
			{2, "input", pkg + "PaymentInput"},
		},
		"ClosePaymentResponse": {
			{1, "execution", pkg + "ExecutionResult"},
			{2, "output", pkg + "PaymentOutput"},
		},
		"RefundRequest": {
			{1, "context", pkg + "OperationContext"},
			{2, "input", pkg + "RefundInput"},
		},
		"RefundResponse": {
			{1, "execution", pkg + "ExecutionResult"},
			{2, "output", pkg + "RefundOutput"},
		},
		"QueryRefundRequest": {
			{1, "context", pkg + "OperationContext"},
			{2, "input", pkg + "RefundInput"},
		},
		"QueryRefundResponse": {
			{1, "execution", pkg + "ExecutionResult"},
			{2, "output", pkg + "RefundOutput"},
		},
		"RouteCallbackRequest": {
			{1, "context", pkg + "OperationContext"},
			{2, "input", pkg + "CallbackPayload"},
		},
		"RouteCallbackResponse": {
			{1, "execution", pkg + "ExecutionResult"},
			{2, "output", pkg + "RouteCallbackOutput"},
		},
		"VerifyCallbackRequest": {
			{1, "context", pkg + "OperationContext"},
			{2, "input", pkg + "VerifyCallbackInput"},
		},
		"VerifyCallbackResponse": {
			{1, "execution", pkg + "ExecutionResult"},
			{2, "output", pkg + "CallbackOutput"},
		},
		"PluginFixture": {
			{1, "path", "string"},
			{2, "content", "bytes"},
		},
		"PluginPackage": {
			{1, "manifest_json", "bytes"},
			{2, "plugin_wasm", "bytes"},
			{3, "fixtures", "repeated " + pkg + "PluginFixture"},
			{4, "package_sha256", "bytes"},
		},
		"PackageDigests": {
			{1, "package_sha256", "bytes"},
			{2, "fixtures_sha256", "bytes"},
			{3, "regression_report_sha256", "bytes"},
		},
		"ApprovalReference": {
			{1, "approval_request_id", "string"},
			{2, "request_digest", "bytes"},
		},
		"ReplayPackageRequest": {
			{1, "package", pkg + "PluginPackage"},
		},
		"ReplayPackageResponse": {
			{1, "digests", pkg + "PackageDigests"},
		},
		"RegisterPackageRequest": {
			{1, "idempotency_key", "string"},
			{2, "package", pkg + "PluginPackage"},
			{3, "approved_digests", pkg + "PackageDigests"},
			{4, "publication", pkg + "ApprovalReference"},
		},
		"RegisterPackageResponse": {
			{1, "digests", pkg + "PackageDigests"},
			{2, "publication", pkg + "ApprovalReference"},
		},
		"RevokePackageRequest": {
			{1, "idempotency_key", "string"},
			{2, "package_sha256", "bytes"},
			{3, "owner_action_id", "string"},
		},
		"RevokePackageResponse": {
			{1, "package_sha256", "bytes"},
			{2, "revoked_at", "google.protobuf.Timestamp"},
		},
		"StoreCredentialsRequest": {
			{1, "idempotency_key", "string"},
			{2, "channel_account_id", "string"},
			{3, "channel_code", "string"},
			{4, "credentials_json", "bytes"},
		},
		"StoreCredentialsResponse": {
			{1, "credentials_ref", "string"},
			{2, "credentials_sha256", "bytes"},
		},
		"GetExchangeRequest": {
			{1, "exchange_id", "string"},
		},
		"GetExchangeResponse": {
			{1, "exchange_id", "string"},
			{2, "canonical_record", "bytes"},
			{3, "evidence_sha256", "bytes"},
		},
	}
	if upstreamFile.Messages().Len() != len(messages) {
		t.Fatalf("unexpected message count %d", upstreamFile.Messages().Len())
	}
	for name, fields := range messages {
		t.Run(string(name), func(t *testing.T) {
			message := upstreamFile.Messages().ByName(name)
			if message == nil || message.Fields().Len() != len(fields) || message.Messages().Len() != 0 || message.Enums().Len() != 0 {
				t.Fatalf("unexpected message shape: %s", name)
			}
			for _, want := range fields {
				field := message.Fields().ByNumber(want.number)
				if field == nil || field.Name() != want.name {
					t.Errorf("field %d: expected %s", want.number, want.name)
					continue
				}
				got := field.Kind().String()
				switch field.Kind() {
				case protoreflect.MessageKind:
					got = string(field.Message().FullName())
				case protoreflect.EnumKind:
					got = string(field.Enum().FullName())
				}
				switch {
				case field.HasOptionalKeyword():
					got = "optional " + got
				case field.IsList():
					got = "repeated " + got
				}
				if got != want.fieldType {
					t.Errorf("%s: got %s, want %s", field.Name(), got, want.fieldType)
				}
				if oneof := field.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() {
					t.Errorf("%s: unexpected oneof %s", field.Name(), oneof.Name())
				}
				wantPresence := field.HasOptionalKeyword() || (field.Kind() == protoreflect.MessageKind && !field.IsList())
				if field.HasPresence() != wantPresence {
					t.Errorf("%s: want presence %t", field.Name(), wantPresence)
				}
			}
		})
	}
}
