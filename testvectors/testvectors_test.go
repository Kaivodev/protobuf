package testvectors_test

import (
	"bytes"
	"cmp"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	collectionv1 "github.com/Kaivodev/protobuf/gen/go/aipay/collection/v1"
	ledgerv1 "github.com/Kaivodev/protobuf/gen/go/aipay/ledger/v1"
	paymentv1 "github.com/Kaivodev/protobuf/gen/go/aipay/payment/v1"
	reconciliationv1 "github.com/Kaivodev/protobuf/gen/go/aipay/reconciliation/v1"
	"github.com/Kaivodev/protobuf/testvectors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// This file re-derives every vector so the published data stays consistent.
// It is test-only on purpose; consumers write their own implementation.

func decodeJSON(t *testing.T, raw []byte, target any) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatal(err)
	}
}

func unhex(t *testing.T, value string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func integer(t *testing.T, value string) uint64 {
	t.Helper()
	if strings.HasPrefix(value, "-") {
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return uint64(v)
	}
	v, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

type writer struct{ b []byte }

func (w *writer) u32(v uint32)   { w.b = binary.BigEndian.AppendUint32(w.b, v) }
func (w *writer) u64(v uint64)   { w.b = binary.BigEndian.AppendUint64(w.b, v) }
func (w *writer) str(v string)   { w.u32(uint32(len(v))); w.b = append(w.b, v...) }
func (w *writer) bytes(v []byte) { w.u32(uint32(len(v))); w.b = append(w.b, v...) }
func (w *writer) raw(v ...byte)  { w.b = append(w.b, v...) }
func sum(parts ...[]byte) []byte {
	h := sha256.New()
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}
func equalHex(a []byte, b string) bool { return hex.EncodeToString(a) == b }

func TestPaymentAccessMaterials(t *testing.T) {
	var doc struct {
		Description string
		Vectors     []struct {
			Name, Encoding, SHA256 string
			Request                json.RawMessage
		}
	}
	decodeJSON(t, testvectors.PaymentAccessMaterials(), &doc)
	if len(doc.Vectors) != 2 {
		t.Fatal("expected historical and per-application materials vectors")
	}
	for _, v := range doc.Vectors {
		t.Run(v.Name, func(t *testing.T) {
			var request paymentv1.SubmitRequest
			if err := protojson.Unmarshal(v.Request, &request); err != nil {
				t.Fatal(err)
			}
			wire, err := (proto.MarshalOptions{Deterministic: true}).Marshal(&request)
			if err != nil {
				t.Fatal(err)
			}
			if !equalHex(wire, v.Encoding) || !equalHex(sum(wire), v.SHA256) {
				t.Fatal("materials encoding or SHA-256 changed")
			}
		})
	}
}

// Re-derive the published Python-generated fixtures using concrete Go requests.
// This is an encoding check, not a claim that IAM or collection enforces it yet.
func TestCollectionCommandV1(t *testing.T) {
	var doc struct {
		Spec, Description string
		Vectors           []struct {
			Name, Method, Encoding, SHA256 string
			Request                        json.RawMessage
		}
	}
	decodeJSON(t, testvectors.CollectionCommandV1(), &doc)
	if doc.Spec != "aipay.collection.command.v1" || len(doc.Vectors) != 4 {
		t.Fatal("unexpected collection vectors")
	}
	seen := map[string]bool{}
	for _, vector := range doc.Vectors {
		t.Run(vector.Name, func(t *testing.T) {
			w := &writer{}
			w.str("aipay.web.command.v1")
			w.str(vector.Method)
			switch vector.Method {
			case collectionv1.CollectionService_CreateSweepJob_FullMethodName:
				var request collectionv1.CreateSweepJobRequest
				if err := protojson.Unmarshal(vector.Request, &request); err != nil {
					t.Fatal(err)
				}
				w.str(request.IdempotencyKey)
				w.str(request.AccountId)
				w.str(request.AddressId)
				w.str(request.AmountAtomic)
				w.str(request.Reason)
			case collectionv1.CollectionService_GetSweepJob_FullMethodName:
				var request collectionv1.GetSweepJobRequest
				if err := protojson.Unmarshal(vector.Request, &request); err != nil {
					t.Fatal(err)
				}
				w.str(request.AccountId)
				switch selector := request.Selector.(type) {
				case *collectionv1.GetSweepJobRequest_SweepId:
					w.u32(2)
					w.str(selector.SweepId)
				case *collectionv1.GetSweepJobRequest_CreateIdempotencyKey:
					w.u32(3)
					w.str(selector.CreateIdempotencyKey)
				default:
					t.Fatal("missing lookup selector")
				}
			case collectionv1.CollectionService_CancelSweepJob_FullMethodName:
				var request collectionv1.CancelSweepJobRequest
				if err := protojson.Unmarshal(vector.Request, &request); err != nil {
					t.Fatal(err)
				}
				w.str(request.IdempotencyKey)
				w.str(request.AccountId)
				w.str(request.SweepId)
				w.str(request.Reason)
			default:
				t.Fatal("unknown method")
			}
			if !equalHex(w.b, vector.Encoding) || !equalHex(sum(w.b), vector.SHA256) {
				t.Fatal("canonical bytes or digest differ from fixed fixture")
			}
			if seen[vector.SHA256] {
				t.Fatal("commands or selectors share a digest")
			}
			seen[vector.SHA256] = true
		})
	}
}

func TestPaymentWebCommandV1(t *testing.T) {
	var doc struct {
		Spec, Description string
		Vectors           []struct {
			Name, Method, Encoding, SHA256 string
			Request                        json.RawMessage
		}
	}
	decodeJSON(t, testvectors.PaymentWebCommandV1(), &doc)
	if doc.Spec != "aipay.web.command.v1" {
		t.Fatal("unexpected command domain")
	}
	var online, offline, requeue, submitEvidence, createRefund bool
	for _, v := range doc.Vectors {
		t.Run(v.Name, func(t *testing.T) {
			var w writer
			w.str(doc.Spec)
			w.str(v.Method)
			switch v.Method {
			case paymentv1.SellerPortalService_SetSellerChannelOnline_FullMethodName:
				var req paymentv1.SetSellerChannelOnlineRequest
				if err := protojson.Unmarshal(v.Request, &req); err != nil {
					t.Fatal(err)
				}
				if req.Online == nil {
					t.Fatal("online must be explicitly present")
				}
				w.str(req.IdempotencyKey)
				w.str(req.ChannelAccountId)
				w.raw(1) // optional scalar is present
				if *req.Online {
					w.raw(1)
					online = true
				} else {
					w.raw(0)
					offline = true
				}
				w.u64(req.ExpectedSellerOnlineVersion)
				w.str(req.Reason)
			case paymentv1.PaymentAdminService_RequeueArchivedWork_FullMethodName:
				var req paymentv1.RequeueArchivedWorkRequest
				if err := protojson.Unmarshal(v.Request, &req); err != nil {
					t.Fatal(err)
				}
				w.str(req.IdempotencyKey)
				w.str(req.Kind)
				w.str(req.ObjectId)
				w.u64(req.ExpectedVersion)
				w.str(req.Reason)
				requeue = true
			case paymentv1.SellerPortalService_SubmitSellerRefundEvidence_FullMethodName:
				var req paymentv1.SubmitSellerRefundEvidenceRequest
				if err := protojson.Unmarshal(v.Request, &req); err != nil {
					t.Fatal(err)
				}
				if req.Note == nil || len(req.Files) == 0 {
					t.Fatal("expected a note and a JPEG upload")
				}
				w.canonicalMessage(t, req.ProtoReflect())
				submitEvidence = true
			case paymentv1.PaymentAdminService_CreatePlatformRefund_FullMethodName:
				var req paymentv1.CreatePlatformRefundRequest
				if err := protojson.Unmarshal(v.Request, &req); err != nil {
					t.Fatal(err)
				}
				w.canonicalMessage(t, req.ProtoReflect())
				createRefund = true
			default:
				t.Fatalf("unreviewed payment command %q", v.Method)
			}
			if !equalHex(w.b, v.Encoding) || !equalHex(sum(w.b), v.SHA256) {
				t.Fatal("canonical bytes or digest differ from fixture")
			}
		})
	}
	if !online || !offline || !requeue || !submitEvidence || !createRefund {
		t.Fatal("missing payment web command fixture")
	}
}

// Encode these fixtures from their generated descriptors, independently of
// the field-by-field encoder used to produce the reference bytes.
func (w *writer) canonicalMessage(t *testing.T, m protoreflect.Message) {
	t.Helper()
	if ts, ok := m.Interface().(*timestamppb.Timestamp); ok {
		instant := ts.AsTime()
		nanos := instant.UnixNano()
		if ts.CheckValid() != nil || ts.Nanos%1000 != 0 || !instant.Equal(time.Unix(0, nanos)) {
			t.Fatal("timestamp must fit whole microseconds in signed Unix nanoseconds")
		}
		w.u64(uint64(nanos))
		return
	}
	fields := m.Descriptor().Fields()
	// Every message used by these fixtures has consecutive field numbers.
	for n := 1; n <= fields.Len(); n++ {
		f := fields.ByNumber(protoreflect.FieldNumber(n))
		if f == nil {
			t.Fatalf("unexpected field numbering on %s", m.Descriptor().FullName())
		}
		if oneof := f.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() {
			selected := m.WhichOneof(oneof)
			if selected == nil {
				t.Fatalf("missing selection for %s", oneof.FullName())
			}
			if f != selected {
				continue
			}
			w.u32(uint32(f.Number()))
		}
		if f.HasOptionalKeyword() {
			if !m.Has(f) {
				w.raw(0)
				continue
			}
			w.raw(1)
		}
		if f.IsList() {
			items := m.Get(f).List()
			w.u32(uint32(items.Len()))
			for i := 0; i < items.Len(); i++ {
				w.canonicalValue(t, f, items.Get(i))
			}
		} else {
			if f.Message() != nil && !m.Has(f) {
				t.Fatalf("missing required %s", f.FullName())
			}
			w.canonicalValue(t, f, m.Get(f))
		}
	}
}

func (w *writer) canonicalValue(t *testing.T, f protoreflect.FieldDescriptor, v protoreflect.Value) {
	t.Helper()
	switch f.Kind() {
	case protoreflect.StringKind:
		w.str(v.String())
	case protoreflect.BytesKind:
		w.bytes(v.Bytes())
	case protoreflect.BoolKind:
		if v.Bool() {
			w.raw(1)
		} else {
			w.raw(0)
		}
	case protoreflect.Int64Kind:
		w.u64(uint64(v.Int()))
	case protoreflect.Uint64Kind:
		w.u64(v.Uint())
	case protoreflect.Uint32Kind:
		w.u32(uint32(v.Uint()))
	case protoreflect.Int32Kind:
		w.u32(uint32(v.Int()))
	case protoreflect.EnumKind:
		w.u32(uint32(v.Enum()))
	case protoreflect.MessageKind:
		w.canonicalMessage(t, v.Message())
	default:
		t.Fatalf("unexpected kind %v on %s", f.Kind(), f.FullName())
	}
}

func TestPaymentSellerRefundV1(t *testing.T) {
	var doc struct {
		Description string
		Evidence    []struct {
			Name, Encoding string
			Material       json.RawMessage
			Digest         string `json:"evidence_sha256"`
		} `json:"evidence_vectors"`
		Approvals []struct {
			Name         string
			Material     json.RawMessage
			RequestBytes string `json:"request_bytes"`
			Digest       string `json:"request_digest"`
		} `json:"approval_vectors"`
	}
	decodeJSON(t, testvectors.PaymentSellerRefundV1(), &doc)
	type evidenceRecord struct {
		material *paymentv1.SellerRefundEvidenceMaterial
		digest   []byte
	}
	records := map[string]evidenceRecord{}
	var threeWithNote, lateWithoutNote bool
	for _, v := range doc.Evidence {
		t.Run(v.Name, func(t *testing.T) {
			var m paymentv1.SellerRefundEvidenceMaterial
			if err := protojson.Unmarshal(v.Material, &m); err != nil {
				t.Fatal(err)
			}
			var w writer
			w.str("aipay.payment.seller_refund_evidence.v1")
			w.canonicalMessage(t, m.ProtoReflect())
			digest := sum(w.b)
			if !equalHex(w.b, v.Encoding) || !equalHex(digest, v.Digest) {
				t.Fatal("evidence bytes or digest differ from fixture")
			}
			records[m.EvidenceId] = evidenceRecord{&m, digest}
			threeWithNote = threeWithNote || m.Note != nil && len(m.Files) == 3
			lateWithoutNote = lateWithoutNote || m.Note == nil && m.Subject.GetLatePaymentId() != ""
		})
	}
	var postRelease, lateReview, failureWithEvidence, failureWithoutEvidence bool
	for _, v := range doc.Approvals {
		t.Run(v.Name, func(t *testing.T) {
			var m paymentv1.PaymentApprovalMaterial
			if err := protojson.Unmarshal(v.Material, &m); err != nil {
				t.Fatal(err)
			}
			var w writer
			w.str("aipay.payment.admin.v1")
			w.canonicalMessage(t, m.ProtoReflect())
			if !equalHex(w.b, v.RequestBytes) || !equalHex(sum([]byte("AIPAY-APPROVAL-V1"), w.b), v.Digest) {
				t.Fatal("approval request bytes or digest differ from fixture")
			}
			switch m.Kind {
			case "seller_refund":
				review := m.GetSellerRefund()
				record, ok := records[review.GetEvidenceId()]
				if !ok || !bytes.Equal(record.digest, review.EvidenceSha256) ||
					!proto.Equal(record.material.Subject, review.Subject) || record.material.Version != review.EvidenceVersion ||
					record.material.SubmittedBy != m.RequestedBy || record.material.UpstreamRefundNo != review.UpstreamRefundNo ||
					!proto.Equal(record.material.RefundedAt, review.RefundedAt) || !proto.Equal(record.material.RefundedAmount, review.RefundedAmount) {
					t.Fatal("review does not bind its evidence fixture")
				}
				postRelease = postRelease || review.Subject.GetRefundId() != "" && review.GetBranch() == ledgerv1.RefundReservationBranch_REFUND_RESERVATION_BRANCH_POST_RELEASE
				lateReview = lateReview || review.Subject.GetLatePaymentId() != "" && review.RefundReservationId == nil &&
					review.Branch == nil && review.ReservationControlVersion == nil && review.OutRefundNo == nil && review.ObjectionWindowSeconds == 0
			case "seller_refund_failure":
				failure := m.GetSellerRefundFailure()
				if failure == nil || (failure.EvidenceId == nil) != (failure.EvidenceSha256 == nil) {
					t.Fatal("incomplete failure material")
				}
				if failure.EvidenceId == nil {
					failureWithoutEvidence = true
				} else {
					record, ok := records[failure.GetEvidenceId()]
					if !ok || !bytes.Equal(record.digest, failure.EvidenceSha256) || !proto.Equal(record.material.Subject, failure.Subject) {
						t.Fatal("failure does not bind its evidence fixture")
					}
					failureWithEvidence = true
				}
			default:
				t.Fatalf("unknown approval kind %q", m.Kind)
			}
		})
	}
	if !threeWithNote || !lateWithoutNote || !postRelease || !lateReview || !failureWithEvidence || !failureWithoutEvidence {
		t.Fatal("missing evidence or approval fixture")
	}
}

func TestPaymentForcedCloseV1(t *testing.T) {
	var doc struct {
		Description string
		Approvals   []struct {
			Name         string
			Material     json.RawMessage
			RequestBytes string `json:"request_bytes"`
			Digest       string `json:"request_digest"`
		} `json:"approval_vectors"`
	}
	decodeJSON(t, testvectors.PaymentForcedCloseV1(), &doc)
	var unbound, bound, multibyteReason bool
	for _, v := range doc.Approvals {
		t.Run(v.Name, func(t *testing.T) {
			var m paymentv1.PaymentApprovalMaterial
			if err := protojson.Unmarshal(v.Material, &m); err != nil {
				t.Fatal(err)
			}
			f := m.GetForcedClose()
			if m.Kind != "forced_close" || f == nil {
				t.Fatal("expected forced_close approval material")
			}
			var w writer
			w.str("aipay.payment.admin.v1")
			w.canonicalMessage(t, m.ProtoReflect())
			if !equalHex(w.b, v.RequestBytes) || !equalHex(sum([]byte("AIPAY-APPROVAL-V1"), w.b), v.Digest) {
				t.Fatal("approval request bytes or digest differ from fixture")
			}
			multibyteReason = multibyteReason || len(f.Reason) > utf8.RuneCountInString(f.Reason)
			switch v.Name {
			case "never-sent-unbound":
				if f.Status != paymentv1.OrderStatus_ORDER_STATUS_UNKNOWN || f.UpstreamOrderId != "" ||
					!f.NeverSent || !proto.Equal(f.EligibleAt, m.RequestedAt) {
					t.Fatal("expected an unsent, unbound order eligible at submission")
				}
				unbound = true
			case "bound-after-ttl":
				if f.Status != paymentv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT || f.UpstreamOrderId == "" ||
					f.NeverSent || !f.EligibleAt.AsTime().Before(m.RequestedAt.AsTime()) {
					t.Fatal("expected a bound order eligible before submission")
				}
				bound = true
			}
		})
	}
	if !unbound || !bound || !multibyteReason {
		t.Fatal("expected never-sent-unbound, bound-after-ttl and a multibyte reason")
	}
}

// JSON decodes data from base64 into the original upload bytes.
type adjustmentUpload struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}

func adjustmentManifestDigest(t *testing.T, uploads []adjustmentUpload, files []*ledgerv1.AdjustmentEvidenceFile) []byte {
	t.Helper()
	if len(uploads) == 0 || len(uploads) > 5 || len(uploads) != len(files) {
		t.Fatalf("got %d uploads and %d manifest entries", len(uploads), len(files))
	}
	var w writer
	w.str("aipay.ledger.adjustment_evidence.v1")
	w.u32(uint32(len(uploads)))
	for i, upload := range uploads {
		size := uint32(len(upload.Data))
		hash := sum(upload.Data)
		file := files[i]
		if file.GetFileNo() != uint32(i+1) || file.GetFileName() != upload.FileName ||
			file.GetContentType() != upload.ContentType || file.GetSizeBytes() != size || !bytes.Equal(file.GetSha256(), hash) {
			t.Fatalf("manifest entry %d differs from upload %q", i+1, upload.FileName)
		}
		w.str(upload.FileName)
		w.str(upload.ContentType)
		w.u32(size)
		w.bytes(hash)
	}
	return sum(w.b)
}

func TestLedgerAdjustmentV1(t *testing.T) {
	var doc struct {
		Description string
		Approvals   []struct {
			Name         string
			Material     json.RawMessage
			Uploads      []adjustmentUpload `json:"evidence_files"`
			RequestBytes string             `json:"request_bytes"`
			Digest       string             `json:"request_digest"`
		} `json:"approval_vectors"`
	}
	decodeJSON(t, testvectors.LedgerAdjustmentV1(), &doc)
	var correction, noCorrection, multibyteReason, debit, credit bool
	for _, v := range doc.Approvals {
		t.Run(v.Name, func(t *testing.T) {
			var m ledgerv1.LedgerApprovalMaterial
			if err := protojson.Unmarshal(v.Material, &m); err != nil {
				t.Fatal(err)
			}
			a := m.GetAdjustment()
			if m.Kind != "adjustment" || a == nil {
				t.Fatal("expected adjustment approval material")
			}
			manifest := adjustmentManifestDigest(t, v.Uploads, a.EvidenceFiles)
			if !bytes.Equal(manifest, a.EvidenceSha256) {
				t.Fatal("manifest digest differs from signed evidence_sha256")
			}
			id := unhex(t, strings.ReplaceAll(m.ApprovalRequestId, "-", ""))
			if len(id) != 16 {
				t.Fatal("approval_request_id must be a UUID")
			}
			var w writer
			w.str("aipay.ledger.adjustment.v1")
			w.raw(id...)
			w.str(m.RequestedBy)
			w.u64(uint64(m.RequestedAt.AsTime().UnixNano()))
			w.u64(uint64(m.ExpiresAt.AsTime().UnixNano()))
			w.str(a.Reason)
			w.bytes(manifest)
			w.u64(a.CorrectsSeq)
			w.u32(uint32(len(a.Lines)))
			balances := map[string]int64{}
			for _, line := range a.Lines {
				if line.Units <= 0 {
					t.Fatal("line units must be positive")
				}
				w.str(line.AccountCode)
				switch line.Direction {
				case ledgerv1.EntryDirection_ENTRY_DIRECTION_DEBIT:
					w.raw('D')
					balances[line.Currency] += line.Units
					debit = true
				case ledgerv1.EntryDirection_ENTRY_DIRECTION_CREDIT:
					w.raw('C')
					balances[line.Currency] -= line.Units
					credit = true
				default:
					t.Fatalf("invalid entry direction %v", line.Direction)
				}
				w.str(line.Currency)
				w.u64(uint64(line.Units))
			}
			for currency, net := range balances {
				if net != 0 {
					t.Fatalf("%s lines do not balance: net %d", currency, net)
				}
			}
			if !equalHex(w.b, v.RequestBytes) || !equalHex(sum([]byte("AIPAY-APPROVAL-V1"), w.b), v.Digest) {
				t.Fatal("canonical request bytes or approval digest differ from fixture")
			}
			multibyteReason = multibyteReason || len(a.Reason) > utf8.RuneCountInString(a.Reason)
			switch v.Name {
			case "four-lines-two-currencies-correction":
				_, usdt := balances["USDT"]
				_, trx := balances["TRX"]
				correction = a.CorrectsSeq != 0 && usdt && trx && len(balances) == 2 && len(v.Uploads) == 2
			case "single-currency-no-correction":
				noCorrection = a.CorrectsSeq == 0 && len(balances) == 1 && len(a.Lines) == 2 && len(v.Uploads) == 1
			}
		})
	}
	if !correction || !noCorrection || !debit || !credit || !multibyteReason {
		t.Fatal("expected both named adjustment cases, debit and credit lines, and a multibyte reason")
	}
}

func TestLedgerCanonicalV1(t *testing.T) {
	var doc struct {
		Spec, Description string
		Chains            []struct {
			Name, Description string
			Transactions      []struct {
				Seq              string `json:"seq"`
				TxID             string `json:"tx_id"`
				Kind             string `json:"kind"`
				IdempotencyKey   string `json:"idempotency_key"`
				OccurredAt       string `json:"occurred_at_unix_nanos"`
				RecordedAt       string `json:"recorded_at_unix_nanos"`
				PricingVersionID string `json:"pricing_version_id"`
				Lines            []struct {
					AccountCode    string `json:"account_code"`
					Direction      string `json:"direction"`
					Currency       string `json:"currency"`
					Units          string `json:"units"`
					USDMicros      string `json:"usd_micros"`
					USDRateVersion string `json:"usd_rate_version"`
				} `json:"lines"`
				PrevHash string `json:"prev_hash"`
				Encoding string `json:"encoding"`
				Hash     string `json:"hash"`
			}
		}
	}
	decodeJSON(t, testvectors.LedgerCanonicalV1(), &doc)
	if doc.Spec != "AIPAY-LEDGER-TX-V1" || len(doc.Chains) == 0 {
		t.Fatal("unexpected ledger vector document")
	}
	for _, chain := range doc.Chains {
		previous := make([]byte, 32)
		for i, tx := range chain.Transactions {
			id := unhex(t, strings.ReplaceAll(tx.TxID, "-", ""))
			if len(id) != 16 {
				t.Fatal("tx_id must be a UUID")
			}
			w := &writer{}
			w.u64(integer(t, tx.Seq))
			w.raw(id...)
			w.str(tx.Kind)
			w.str(tx.IdempotencyKey)
			w.u64(integer(t, tx.OccurredAt))
			w.u64(integer(t, tx.RecordedAt))
			w.str(tx.PricingVersionID)
			w.u32(uint32(len(tx.Lines)))
			for _, line := range tx.Lines {
				if line.Direction != "D" && line.Direction != "C" {
					t.Fatal("direction must be D or C")
				}
				w.str(line.AccountCode)
				w.raw(line.Direction[0])
				w.str(line.Currency)
				w.u64(integer(t, line.Units))
				w.u64(integer(t, line.USDMicros))
				w.str(line.USDRateVersion)
			}
			prev := unhex(t, tx.PrevHash)
			if chain.Name == "genesis_chain" && !bytes.Equal(prev, previous) {
				t.Fatalf("%s[%d] does not continue the chain", chain.Name, i)
			}
			hash := sum([]byte("AIPAY-LEDGER-TX-V1"), prev, w.b)
			if !equalHex(w.b, tx.Encoding) || !equalHex(hash, tx.Hash) {
				t.Fatalf("%s[%d] encoding or hash differs", chain.Name, i)
			}
			previous = hash
		}
	}
}

type envelope struct {
	EventID       string `json:"event_id"`
	Producer      string `json:"producer"`
	Stream        string `json:"stream"`
	Sequence      string `json:"sequence"`
	EventType     string `json:"event_type"`
	SchemaVersion string `json:"schema_version"`
	OccurredAt    string `json:"occurred_at_unix_nanos"`
	RecordedAt    string `json:"recorded_at_unix_nanos"`
	Payload       string `json:"payload"`
	PayloadDigest string `json:"payload_digest"`
	PrevDigest    string `json:"prev_digest"`
	KeyID         string `json:"key_id"`
	Signature     string `json:"signature"`
}

func TestEventSigningV1(t *testing.T) {
	var doc struct {
		Spec, Description, Producer, Stream string
		PublicKeys                          []struct {
			KeyID   string            `json:"key_id"`
			JWK     map[string]string `json:"jwk"`
			SPKIDER string            `json:"spki_der"`
		} `json:"public_keys"`
		Events []struct {
			Envelope     envelope `json:"envelope"`
			SigningInput string   `json:"signing_input"`
			EventDigest  string   `json:"event_digest"`
		}
		Invalid []struct {
			Name, Description string
			Cursor            struct {
				Sequence    string `json:"sequence"`
				EventDigest string `json:"event_digest"`
			} `json:"cursor"`
			Envelope envelope `json:"envelope"`
		}
	}
	decodeJSON(t, testvectors.EventSigningV1(), &doc)
	keys := map[string]*ecdsa.PublicKey{}
	for _, key := range doc.PublicKeys {
		parsed, err := x509.ParsePKIXPublicKey(unhex(t, key.SPKIDER))
		if err != nil {
			t.Fatal(err)
		}
		pub := parsed.(*ecdsa.PublicKey)
		raw, _ := pub.Bytes()
		if key.JWK["kty"] != "EC" || key.JWK["crv"] != "P-256" || key.JWK["x"] != base64.RawURLEncoding.EncodeToString(raw[1:33]) || key.JWK["y"] != base64.RawURLEncoding.EncodeToString(raw[33:]) {
			t.Fatal("JWK and SPKI forms differ")
		}
		keys[key.KeyID] = pub
	}
	check := func(e envelope, sequence uint64, digest []byte) ([]byte, string) {
		if e.Producer != doc.Producer || e.Stream != doc.Stream {
			return nil, "producer"
		}
		if integer(t, e.Sequence) != sequence+1 {
			return nil, "sequence"
		}
		if !bytes.Equal(unhex(t, e.PrevDigest), digest) {
			return nil, "prev_digest"
		}
		if !equalHex(sum(unhex(t, e.Payload)), e.PayloadDigest) {
			return nil, "payload_digest"
		}
		w := &writer{b: []byte("AIPAY-EVENT-V1")}
		w.str(e.Producer)
		w.str(e.Stream)
		w.u64(integer(t, e.Sequence))
		w.str(e.EventID)
		w.str(e.EventType)
		w.u64(integer(t, e.SchemaVersion))
		w.u64(integer(t, e.OccurredAt))
		w.bytes(unhex(t, e.PayloadDigest))
		w.bytes(unhex(t, e.PrevDigest))
		key, ok := keys[e.KeyID]
		if !ok {
			return nil, "key_id"
		}
		if !ecdsa.VerifyASN1(key, sum(w.b), unhex(t, e.Signature)) {
			return nil, "signature"
		}
		return w.b, ""
	}
	sequence, digest := uint64(0), make([]byte, 32)
	for i, event := range doc.Events {
		input, failure := check(event.Envelope, sequence, digest)
		if failure != "" || !equalHex(input, event.SigningInput) || !equalHex(sum(input), event.EventDigest) {
			t.Fatalf("event %d: %s", i, failure)
		}
		sequence, digest = sequence+1, sum(input)
	}
	if len(doc.Invalid) == 0 {
		t.Fatal("no invalid event vectors")
	}
	for _, invalid := range doc.Invalid {
		if _, failure := check(invalid.Envelope, integer(t, invalid.Cursor.Sequence), unhex(t, invalid.Cursor.EventDigest)); failure == "" {
			t.Fatalf("%s verified", invalid.Name)
		}
	}
}

// counterAdvances accepts 0 after 0 (synced passkeys without a counter) and
// otherwise requires a strictly increasing counter.
func counterAdvances(counter, stored uint32) bool {
	return counter > stored || (counter == 0 && stored == 0)
}

func TestApprovalV1(t *testing.T) {
	type assertion struct {
		Name, Description string
		StoredSignCount   uint32 `json:"stored_sign_count"`
		Assertion         struct {
			CredentialID      string `json:"credential_id"`
			AuthenticatorData string `json:"authenticator_data"`
			ClientDataJSON    string `json:"client_data_json"`
			Signature         string `json:"signature"`
		}
	}
	var doc struct {
		Spec, Description string
		RPID              string `json:"rp_id"`
		Origin            string
		RequestBytes      string `json:"request_bytes"`
		RequestDigest     string `json:"request_digest"`
		Credential        struct {
			CredentialID  string `json:"credential_id"`
			COSEPublicKey string `json:"cose_public_key"`
		}
		Valid, Invalid []assertion
	}
	decodeJSON(t, testvectors.ApprovalV1(), &doc)
	digest := sum([]byte("AIPAY-APPROVAL-V1"), unhex(t, doc.RequestBytes))
	if !equalHex(digest, doc.RequestDigest) {
		t.Fatal("request digest differs")
	}
	// The fixed shape of a COSE EC2 P-256 ES256 key: {1: 2, 3: -7, -1: 1, -2: x, -3: y}.
	cose := unhex(t, doc.Credential.COSEPublicKey)
	if len(cose) != 77 || !bytes.Equal(cose[:10], unhex(t, "a5010203262001215820")) || !bytes.Equal(cose[42:45], unhex(t, "225820")) {
		t.Fatal("unexpected COSE key layout")
	}
	pub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), append([]byte{4}, append(cose[10:42:42], cose[45:]...)...))
	if err != nil {
		t.Fatal(err)
	}
	rpHash := sha256.Sum256([]byte(doc.RPID))
	verify := func(a assertion) string {
		auth, clientData := unhex(t, a.Assertion.AuthenticatorData), unhex(t, a.Assertion.ClientDataJSON)
		var client struct {
			Type, Challenge, Origin string
			CrossOrigin             bool `json:"crossOrigin"`
		}
		if err := json.Unmarshal(clientData, &client); err != nil || len(auth) < 37 {
			return "malformed"
		}
		challenge, err := base64.RawURLEncoding.DecodeString(client.Challenge)
		switch {
		case a.Assertion.CredentialID != doc.Credential.CredentialID:
			return "credential"
		case client.Type != "webauthn.get":
			return "type"
		case err != nil || !bytes.Equal(challenge, digest):
			return "challenge"
		case client.Origin != doc.Origin || client.CrossOrigin:
			return "origin"
		case !bytes.Equal(auth[:32], rpHash[:]):
			return "rp_id"
		case auth[32]&0x01 == 0 || auth[32]&0x04 == 0:
			return "flags"
		case !counterAdvances(binary.BigEndian.Uint32(auth[33:37]), a.StoredSignCount):
			return "counter"
		case !ecdsa.VerifyASN1(pub, sum(auth, sum(clientData)), unhex(t, a.Assertion.Signature)):
			return "signature"
		}
		return ""
	}
	for _, a := range doc.Valid {
		if failure := verify(a); failure != "" {
			t.Fatalf("%s: %s", a.Name, failure)
		}
	}
	for _, a := range doc.Invalid {
		if verify(a) == "" {
			t.Fatalf("%s verified", a.Name)
		}
		// Invalid cases differ from the valid one only in the named property, so
		// the signature itself must still verify.
		auth, clientData := unhex(t, a.Assertion.AuthenticatorData), unhex(t, a.Assertion.ClientDataJSON)
		if !ecdsa.VerifyASN1(pub, sum(auth, sum(clientData)), unhex(t, a.Assertion.Signature)) {
			t.Fatalf("%s carries a bad signature", a.Name)
		}
	}
}

func TestLedgerCheckpointV1(t *testing.T) {
	type balance struct {
		AccountCode string `json:"account_code"`
		Units       string `json:"units"`
		USDMicros   string `json:"usd_micros"`
	}
	type row struct {
		GLCode       string `json:"gl_code"`
		DimensionKey string `json:"dimension_key"`
		Currency     string `json:"currency"`
		OpeningUnits string `json:"opening_units"`
		DebitUnits   string `json:"debit_units"`
		CreditUnits  string `json:"credit_units"`
		ClosingUnits string `json:"closing_units"`
		OpeningUSD   string `json:"opening_usd_micros"`
		DebitUSD     string `json:"debit_usd_micros"`
		CreditUSD    string `json:"credit_usd_micros"`
		ClosingUSD   string `json:"closing_usd_micros"`
	}
	type checkpoint struct {
		ID                 string `json:"id"`
		Kind               string `json:"kind"`
		FromSeq            string `json:"from_seq"`
		ToSeq              string `json:"to_seq"`
		HeadHash           string `json:"head_hash"`
		MerkleRoot         string `json:"merkle_root"`
		BalancesDigest     string `json:"balances_digest"`
		TrialBalanceDigest string `json:"trial_balance_digest"`
		BusinessDate       string `json:"business_date"`
		PeriodID           string `json:"period_id"`
		PrevDigest         string `json:"prev_checkpoint_digest"`
		KeyID              string `json:"key_id"`
		Signature          string `json:"signature"`
	}
	var doc struct {
		Spec, Description string
		Balances          []struct {
			Name, Description, Seq string
			Balances               []balance
			Input, Digest          string
		}
		TrialBalances []struct {
			Name, Description string
			PeriodID          string `json:"period_id"`
			Rows              []row
			Input, Digest     string
		} `json:"trial_balances"`
		Merkle []struct {
			Leaves            []string
			Root, Description string
		}
		PublicKeys []struct {
			KeyID   string            `json:"key_id"`
			JWK     map[string]string `json:"jwk"`
			SPKIDER string            `json:"spki_der"`
		} `json:"public_keys"`
		Checkpoints []struct {
			Checkpoint   checkpoint
			SigningInput string `json:"signing_input"`
			Digest       string
		}
		Invalid []struct {
			Name, Description string
			Checkpoint        checkpoint
		} `json:"invalid_checkpoints"`
	}
	decodeJSON(t, testvectors.LedgerCheckpointV1(), &doc)
	for _, c := range doc.Balances {
		rows := slices.Clone(c.Balances)
		slices.SortFunc(rows, func(a, b balance) int { return strings.Compare(a.AccountCode, b.AccountCode) })
		w := &writer{b: []byte("AIPAY-LEDGER-BALANCES-V1")}
		w.u64(integer(t, c.Seq))
		w.u64(uint64(len(rows)))
		for _, r := range rows {
			w.str(r.AccountCode)
			w.u64(integer(t, r.Units))
			w.u64(integer(t, r.USDMicros))
		}
		if !equalHex(w.b, c.Input) || !equalHex(sum(w.b), c.Digest) {
			t.Fatalf("balances %s differ", c.Name)
		}
	}
	for _, c := range doc.TrialBalances {
		rows := slices.Clone(c.Rows)
		slices.SortFunc(rows, func(a, b row) int {
			return strings.Compare(a.GLCode+"\x00"+a.DimensionKey+"\x00"+a.Currency, b.GLCode+"\x00"+b.DimensionKey+"\x00"+b.Currency)
		})
		w := &writer{b: []byte("AIPAY-LEDGER-TRIAL-BALANCE-V1")}
		w.str(c.PeriodID)
		w.u64(uint64(len(rows)))
		for _, r := range rows {
			w.str(r.GLCode)
			w.str(r.DimensionKey)
			w.str(r.Currency)
			for _, v := range []string{r.OpeningUnits, r.DebitUnits, r.CreditUnits, r.ClosingUnits, r.OpeningUSD, r.DebitUSD, r.CreditUSD, r.ClosingUSD} {
				w.u64(integer(t, v))
			}
		}
		if !equalHex(w.b, c.Input) || !equalHex(sum(w.b), c.Digest) {
			t.Fatalf("trial balance %s differs", c.Name)
		}
	}
	var merkle func(leaves [][]byte) []byte
	merkle = func(leaves [][]byte) []byte {
		switch len(leaves) {
		case 0:
			return sum(nil)
		case 1:
			return sum([]byte{0}, leaves[0])
		}
		k := 1
		for k*2 < len(leaves) {
			k *= 2
		}
		return sum([]byte{1}, merkle(leaves[:k]), merkle(leaves[k:]))
	}
	for i, c := range doc.Merkle {
		var leaves [][]byte
		for _, l := range c.Leaves {
			leaves = append(leaves, unhex(t, l))
		}
		if !equalHex(merkle(leaves), c.Root) {
			t.Fatalf("merkle case %d differs", i)
		}
	}
	keys := map[string]*ecdsa.PublicKey{}
	for _, key := range doc.PublicKeys {
		parsed, err := x509.ParsePKIXPublicKey(unhex(t, key.SPKIDER))
		if err != nil {
			t.Fatal(err)
		}
		keys[key.KeyID] = parsed.(*ecdsa.PublicKey)
	}
	input := func(c checkpoint) []byte {
		w := &writer{b: []byte("AIPAY-LEDGER-CHECKPOINT-V1")}
		w.u64(integer(t, c.ID))
		w.str(c.Kind)
		w.u64(integer(t, c.FromSeq))
		w.u64(integer(t, c.ToSeq))
		w.bytes(unhex(t, c.HeadHash))
		w.bytes(unhex(t, c.MerkleRoot))
		w.bytes(unhex(t, c.BalancesDigest))
		w.bytes(unhex(t, c.TrialBalanceDigest))
		w.str(c.BusinessDate)
		w.str(c.PeriodID)
		w.bytes(unhex(t, c.PrevDigest))
		return w.b
	}
	verify := func(c checkpoint, prev []byte) string {
		key, ok := keys[c.KeyID]
		switch {
		case !ok:
			return "key_id"
		case !bytes.Equal(unhex(t, c.PrevDigest), prev):
			return "prev_digest"
		case !ecdsa.VerifyASN1(key, sum(input(c)), unhex(t, c.Signature)):
			return "signature"
		}
		return ""
	}
	prev := make([]byte, 32)
	digests := map[string][]byte{"0": prev}
	for _, c := range doc.Checkpoints {
		in := input(c.Checkpoint)
		if !equalHex(in, c.SigningInput) || !equalHex(sum(in), c.Digest) {
			t.Fatalf("checkpoint %s input or digest differs", c.Checkpoint.ID)
		}
		if failure := verify(c.Checkpoint, prev); failure != "" {
			t.Fatalf("checkpoint %s: %s", c.Checkpoint.ID, failure)
		}
		prev = sum(in)
		digests[c.Checkpoint.ID] = prev
	}
	if len(doc.Invalid) == 0 {
		t.Fatal("no invalid checkpoint vectors")
	}
	for _, c := range doc.Invalid {
		previous, _ := strconv.ParseUint(c.Checkpoint.ID, 10, 64)
		if verify(c.Checkpoint, digests[strconv.FormatUint(previous-1, 10)]) == "" {
			t.Fatalf("%s verified", c.Name)
		}
	}
}

type coverageAddressKey struct{ chain, address string }

func (k coverageAddressKey) compare(other coverageAddressKey) int {
	return cmp.Or(strings.Compare(k.chain, other.chain), strings.Compare(k.address, other.address))
}

func TestReconciliationCoverageV1(t *testing.T) {
	var doc struct {
		Description string
		Evidence    []struct {
			Name, Evidence string
			Digest         string `json:"evidence_sha256"`
		} `json:"evidence_vectors"`
		Registries []struct {
			Name    string
			Entries []json.RawMessage
			Version string `json:"registry_version"`
		} `json:"address_registry_vectors"`
		Approvals []struct {
			Name          string
			Material      json.RawMessage
			RequestBytes  string `json:"request_bytes"`
			RequestDigest string `json:"request_digest"`
		} `json:"approval_vectors"`
		Revocations []struct {
			Name, Reason, Challenge string
			UserID                  string `json:"user_id"`
			Key                     string `json:"idempotency_key"`
			AttestationID           string `json:"attestation_id"`
			RequestDigest           string `json:"request_digest"`
		} `json:"revocation_vectors"`
	}
	decodeJSON(t, testvectors.ReconciliationCoverageV1(), &doc)
	evidence := map[string]bool{}
	var multibyte, emptyRegistry bool
	for _, v := range doc.Evidence {
		t.Run(v.Name, func(t *testing.T) {
			if !utf8.ValidString(v.Evidence) || len(v.Evidence) < 1 || len(v.Evidence) > 4096 {
				t.Fatal("invalid UTF-8 evidence or byte length")
			}
			var w writer
			w.str("aipay.reconciliation.coverage_evidence.v1")
			w.bytes([]byte(v.Evidence))
			if !equalHex(sum(w.b), v.Digest) {
				t.Fatal("evidence digest differs")
			}
			evidence[v.Digest] = true
			multibyte = multibyte || len(v.Evidence) != utf8.RuneCountInString(v.Evidence)
		})
	}
	registries := map[string]map[coverageAddressKey]reconciliationv1.CoverageAddressRole{}
	for _, v := range doc.Registries {
		t.Run(v.Name, func(t *testing.T) {
			var w writer
			w.str("aipay.reconciliation.coverage_addresses.v1")
			w.u32(uint32(len(v.Entries)))
			entries := map[coverageAddressKey]reconciliationv1.CoverageAddressRole{}
			var previous coverageAddressKey
			for i, raw := range v.Entries {
				var entry reconciliationv1.CoverageAddress
				if err := protojson.Unmarshal(raw, &entry); err != nil {
					t.Fatal(err)
				}
				key := coverageAddressKey{entry.Chain, entry.Address}
				if i > 0 && previous.compare(key) >= 0 {
					t.Fatal("registry entries must be strictly ordered")
				}
				previous = key
				entries[key] = entry.Role
				w.str(entry.Chain)
				w.str(entry.Address)
				w.u32(uint32(entry.Role))
			}
			if !equalHex(sum(w.b), v.Version) {
				t.Fatal("registry version differs")
			}
			registries[v.Version] = entries
			emptyRegistry = emptyRegistry || len(entries) == 0
		})
	}
	digests := map[string]string{} // Attestation ID to verified approval digest.
	roles := map[reconciliationv1.CoverageAddressRole]bool{}
	results := map[reconciliationv1.CoverageResult]bool{}
	var withCheckpoint, withoutCheckpoint, withNote, withoutNote, capped, inflows, outflows, unclassified bool
	for _, v := range doc.Approvals {
		t.Run(v.Name, func(t *testing.T) {
			var h reconciliationv1.ReconciliationApprovalMaterial
			if err := protojson.Unmarshal(v.Material, &h); err != nil {
				t.Fatal(err)
			}
			m := h.GetCoverageAttestation()
			if h.Kind != "coverage_attestation" || m == nil || m.PolicyVersion != "coverage-manual-v1" {
				t.Fatal("unexpected approval kind, branch or policy")
			}
			var w writer
			w.str("aipay.reconciliation.admin.v1")
			w.canonicalMessage(t, h.ProtoReflect())
			if !equalHex(w.b, v.RequestBytes) || !equalHex(sum([]byte("AIPAY-APPROVAL-V1"), w.b), v.RequestDigest) {
				t.Fatal("approval request bytes or digest differ")
			}
			registry := registries[hex.EncodeToString(m.AddressRegistryVersion)]
			for _, obs := range m.Observations {
				role, ok := registry[coverageAddressKey{obs.Chain, obs.Address}]
				if !ok || role != obs.Role || !evidence[hex.EncodeToString(obs.EvidenceSha256)] {
					t.Fatal("observation does not match its registry or evidence")
				}
				roles[role] = true
			}
			binds := checkCoverageCalculation(t, m)
			capped = capped || binds
			// Every reconciliation approval request expires exactly 24 hours after it
			// was made; the attestation's own expires_at is checked at approval time.
			if !h.ExpiresAt.AsTime().Equal(h.RequestedAt.AsTime().Add(24 * time.Hour)) {
				t.Fatal("approval expiry is not 24 hours after the request")
			}
			withCheckpoint = withCheckpoint || m.Ledger.CheckpointId != nil
			withoutCheckpoint = withoutCheckpoint || m.Ledger.CheckpointId == nil
			withNote = withNote || m.Note != nil
			withoutNote = withoutNote || m.Note == nil
			inflows = inflows || len(m.UnrecordedInflows) > 0
			outflows = outflows || len(m.WindowOutflows) > 0
			unclassified = unclassified || len(m.Ledger.Unclassified) > 0 && len(m.Ledger.Liabilities) > 0
			results[m.Result] = true
			digests[m.AttestationId] = v.RequestDigest
		})
	}
	for _, v := range doc.Revocations {
		t.Run(v.Name, func(t *testing.T) {
			if digest, ok := digests[v.AttestationID]; !ok || digest != v.RequestDigest {
				t.Fatal("revocation does not bind a verified approval")
			}
			var w writer
			w.str("aipay.reconciliation.coverage.revoke.v1")
			w.str(v.UserID)
			w.str(v.Key)
			w.str(v.AttestationID)
			w.bytes(unhex(t, v.RequestDigest))
			w.str(v.Reason)
			if !equalHex(sum([]byte("AIPAY-APPROVAL-V1"), w.b), v.Challenge) {
				t.Fatal("revocation challenge differs")
			}
		})
	}
	if !roles[reconciliationv1.CoverageAddressRole_COVERAGE_ADDRESS_ROLE_COLD] ||
		!roles[reconciliationv1.CoverageAddressRole_COVERAGE_ADDRESS_ROLE_HOT] ||
		!roles[reconciliationv1.CoverageAddressRole_COVERAGE_ADDRESS_ROLE_DEPOSIT] ||
		len(results) != 3 || !capped || !inflows || !outflows || !unclassified {
		t.Error("missing coverage result, wallet role, cap, transfer or liability example")
	}
	if !multibyte || !emptyRegistry || len(doc.Revocations) == 0 || !withCheckpoint || !withoutCheckpoint || !withNote || !withoutNote {
		t.Error("missing UTF-8 evidence, empty registry, revocation or optional field example")
	}
}

// Check each intermediate amount before using it in another sum or difference.
func checkedCoverageUnits(t *testing.T, n *big.Int) int64 {
	t.Helper()
	if !n.IsInt64() {
		t.Fatal("coverage amount exceeds int64")
	}
	return n.Int64()
}

func coverageTransferUnits(t *testing.T, transfers []*reconciliationv1.CoverageTransfer, observed map[coverageAddressKey]bool) int64 {
	t.Helper()
	var total big.Int
	for i, tr := range transfers {
		if !observed[coverageAddressKey{tr.Chain, tr.Address}] || tr.Units <= 0 {
			t.Fatal("transfer requires an observed address and positive amount")
		}
		if i > 0 {
			prev := transfers[i-1]
			if cmp.Or(strings.Compare(prev.Chain, tr.Chain), strings.Compare(prev.TransactionId, tr.TransactionId),
				cmp.Compare(prev.EventIndex, tr.EventIndex)) >= 0 {
				t.Fatal("transfer keys must be strictly ordered")
			}
		}
		total.Add(&total, big.NewInt(tr.Units))
	}
	return checkedCoverageUnits(t, &total)
}

func checkCoverageCalculation(t *testing.T, m *reconciliationv1.CoverageAttestationMaterial) bool {
	t.Helper()
	if len(m.Observations) == 0 || len(m.Observations) > 200 || len(m.UnrecordedInflows) > 200 || len(m.WindowOutflows) > 200 {
		t.Fatal("observation or transfer count out of range")
	}
	observed := map[coverageAddressKey]bool{}
	var previous coverageAddressKey
	var assets, liability big.Int
	cutoff := m.Observations[0].BlockTime.AsTime()
	for i, obs := range m.Observations {
		key := coverageAddressKey{obs.Chain, obs.Address}
		if i > 0 && previous.compare(key) >= 0 {
			t.Fatal("observations must be strictly ordered")
		}
		previous = key
		observed[key] = true
		if obs.Chain != "TRON" || obs.TokenContract != "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" ||
			obs.Units < 0 || obs.RawBalance != strconv.FormatInt(obs.Units, 10) {
			t.Fatal("expected canonical TRON USDT balance")
		}
		if obs.BlockTime.AsTime().Before(cutoff) {
			cutoff = obs.BlockTime.AsTime()
		}
		assets.Add(&assets, big.NewInt(obs.Units))
	}
	if !m.CutoffAt.AsTime().Equal(cutoff) || checkedCoverageUnits(t, &assets) != m.ObservedUnits {
		t.Fatal("observation cutoff or total differs")
	}
	types := map[string]bool{}
	for _, rows := range [][]*reconciliationv1.CoverageTypeBalance{m.Ledger.Liabilities, m.Ledger.Unclassified} {
		for i, row := range rows {
			if types[row.AccountType] || i > 0 && rows[i-1].AccountType >= row.AccountType {
				t.Fatal("liability types must be unique and each list strictly ordered")
			}
			types[row.AccountType] = true
			liability.Add(&liability, big.NewInt(row.Units))
			checkedCoverageUnits(t, &liability)
		}
	}
	if liability.Int64() != m.Ledger.UserLiabilityUnits {
		t.Fatal("ledger liability total differs")
	}
	for _, transfers := range [][]*reconciliationv1.CoverageTransfer{m.WindowOutflows, m.UnrecordedInflows} {
		assets.Sub(&assets, big.NewInt(coverageTransferUnits(t, transfers, observed)))
		checkedCoverageUnits(t, &assets)
	}
	counted := min(assets.Int64(), m.Ledger.CustomerWalletUnits)
	margin := big.NewInt(counted)
	margin.Sub(margin, big.NewInt(m.Ledger.UnmatchedReceiptUnits))
	checkedCoverageUnits(t, margin)
	margin.Sub(margin, &liability)
	if checkedCoverageUnits(t, margin) != m.MarginUnits || counted != m.CountedAssetUnits {
		t.Fatal("counted assets or margin differ")
	}
	scaledMargin := new(big.Int).Mul(margin, big.NewInt(10000))
	target := new(big.Int).Mul(&liability, big.NewInt(int64(m.BufferBps)))
	want := reconciliationv1.CoverageResult_COVERAGE_RESULT_COVERED
	if margin.Sign() < 0 {
		want = reconciliationv1.CoverageResult_COVERAGE_RESULT_NOT_COVERED
	} else if liability.Sign() > 0 && scaledMargin.Cmp(target) < 0 {
		want = reconciliationv1.CoverageResult_COVERAGE_RESULT_COVERED_LOW_MARGIN
	}
	if want != m.Result {
		t.Fatalf("computed %s, material declares %s", want, m.Result)
	}
	return counted < assets.Int64()
}
