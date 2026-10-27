package testvectors_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/aipay-dev/protobuf/testvectors"
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
