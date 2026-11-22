// Package testvectors publishes the contract test vectors as data only.
//
// It deliberately contains no encoder or verifier: settlement and
// reconciliation implement journal hash verification independently of the
// ledger code (decision A16), and every receiver implements its own event and
// approval checks, then tests them against these vectors. Each function returns
// a fresh copy of the JSON document; the format is described in its
// "description" field.
package testvectors

import (
	"bytes"
	_ "embed"
)

var (
	//go:embed ledger_canonical_v1.json
	ledgerCanonicalV1 []byte
	//go:embed event_signing_v1.json
	eventSigningV1 []byte
	//go:embed approval_v1.json
	approvalV1 []byte
	//go:embed ledger_checkpoint_v1.json
	ledgerCheckpointV1 []byte
	//go:embed collection_command_v1.json
	collectionCommandV1 []byte
	//go:embed payment_access_materials.json
	paymentAccessMaterials []byte
	//go:embed payment_web_command_v1.json
	paymentWebCommandV1 []byte
)

// LedgerCanonicalV1 holds journal transactions with their canonical encoding V1
// and hashes, including a chain from the genesis prev_hash.
func LedgerCanonicalV1() []byte { return bytes.Clone(ledgerCanonicalV1) }

// EventSigningV1 holds signed event envelopes with their AIPAY-EVENT-V1
// signing input, event digests, the producer public key, and envelopes that
// must stop the stream.
func EventSigningV1() []byte { return bytes.Clone(eventSigningV1) }

// LedgerCheckpointV1 holds balances digests, trial balance digests, RFC 6962
// Merkle roots and signed checkpoints over the LedgerCanonicalV1 genesis chain,
// with checkpoints that must be rejected.
func LedgerCheckpointV1() []byte { return bytes.Clone(ledgerCheckpointV1) }

// ApprovalV1 holds an approval request digest and WebAuthn assertions over it,
// one valid and the rest invalid.
func ApprovalV1() []byte { return bytes.Clone(approvalV1) }

// CollectionCommandV1 holds canonical command bytes and hashes for collection
// create/get/cancel, including both query selectors. Data only, not validation.
func CollectionCommandV1() []byte { return bytes.Clone(collectionCommandV1) }

// PaymentAccessMaterials holds normalized revisions and their digest inputs
// before and after per-application notification origins were introduced.
func PaymentAccessMaterials() []byte { return bytes.Clone(paymentAccessMaterials) }

// PaymentWebCommandV1 holds canonical command bytes and hashes for seller
// channel switches and archived work requeues.
func PaymentWebCommandV1() []byte { return bytes.Clone(paymentWebCommandV1) }
