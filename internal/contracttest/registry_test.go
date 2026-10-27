// Package contracttest checks invariants of the published contract data.
package contracttest

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"regexp"
	"testing"

	commonv1 "github.com/aipay-dev/protobuf/gen/go/aipay/common/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// The contract's currency table; changing it must change this test too.
var contractCurrencies = map[string]uint32{"CNY": 2, "USDT": 6, "TRX": 6, "BNB": 9}

func values(t *testing.T, enum protoreflect.EnumDescriptor, ext protoreflect.ExtensionType) map[string]proto.Message {
	t.Helper()
	result := map[string]proto.Message{}
	for i := 0; i < enum.Values().Len(); i++ {
		value := enum.Values().Get(i)
		has := proto.HasExtension(value.Options(), ext)
		if value.Number() == 0 {
			if has {
				t.Fatalf("%s must not carry registry data", value.FullName())
			}
			continue
		}
		if !has {
			t.Fatalf("%s lacks registry data", value.FullName())
		}
		result[string(value.Name())] = proto.GetExtension(value.Options(), ext).(proto.Message)
	}
	return result
}

func TestCurrencyRegistry(t *testing.T) {
	codes := map[string]uint32{}
	for name, message := range values(t, commonv1.Currency(0).Descriptor(), commonv1.E_CurrencyInfo) {
		info := message.(*commonv1.CurrencyInfo)
		if name != "CURRENCY_"+info.GetCode() {
			t.Fatalf("%s carries code %q", name, info.GetCode())
		}
		codes[info.GetCode()] = info.GetMinorUnitExponent()
	}
	if len(codes) != len(contractCurrencies) {
		t.Fatalf("registry %v differs from contract %v", codes, contractCurrencies)
	}
	for code, exponent := range contractCurrencies {
		if codes[code] != exponent {
			t.Fatalf("%s exponent %d, contract says %d", code, codes[code], exponent)
		}
	}
}

func TestChainAndAssetRegistry(t *testing.T) {
	chains := map[commonv1.Chain]*commonv1.ChainInfo{}
	for name, message := range values(t, commonv1.Chain(0).Descriptor(), commonv1.E_ChainInfo) {
		info := message.(*commonv1.ChainInfo)
		if name != "CHAIN_"+info.GetCode() {
			t.Fatalf("%s carries code %q", name, info.GetCode())
		}
		if _, ok := contractCurrencies[info.GetGasCurrency()]; !ok || info.GetGasChainDecimals() < contractCurrencies[info.GetGasCurrency()] {
			t.Fatalf("%s gas asset %q is not a registered currency with enough chain decimals", name, info.GetGasCurrency())
		}
		chains[commonv1.Chain(commonv1.Chain_value[name])] = info
	}
	if chains[commonv1.Chain_CHAIN_TRON].GetGasCurrency() != "TRX" || chains[commonv1.Chain_CHAIN_BSC].GetGasCurrency() != "BNB" {
		t.Fatal("gas assets differ from the contract")
	}
	want := map[string]struct {
		chain    commonv1.Chain
		decimals uint32
	}{"REGISTERED_ASSET_TRON_USDT": {commonv1.Chain_CHAIN_TRON, 6}, "REGISTERED_ASSET_BSC_USDT": {commonv1.Chain_CHAIN_BSC, 18}}
	assets := values(t, commonv1.RegisteredAsset(0).Descriptor(), commonv1.E_AssetInfo)
	if len(assets) != len(want) {
		t.Fatalf("registered assets %v differ from the contract", assets)
	}
	contracts := map[string]bool{}
	for name, message := range assets {
		info := message.(*commonv1.AssetInfo)
		expected := want[name]
		if info.GetChain() != expected.chain || info.GetCurrency() != "USDT" || info.GetChainDecimals() != expected.decimals || info.GetChainDecimals() < contractCurrencies["USDT"] {
			t.Fatalf("%s = %v", name, info)
		}
		if contracts[info.GetContractAddress()] {
			t.Fatalf("%s reuses a contract address", name)
		}
		contracts[info.GetContractAddress()] = true
		switch info.GetChain() {
		case commonv1.Chain_CHAIN_TRON:
			if !validTronAddress(info.GetContractAddress()) {
				t.Fatalf("%s has an invalid TRON address", name)
			}
		case commonv1.Chain_CHAIN_BSC:
			if !regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`).MatchString(info.GetContractAddress()) {
				t.Fatalf("%s has an invalid BSC address", name)
			}
		default:
			t.Fatalf("%s uses an unregistered chain", name)
		}
	}
}

// validTronAddress checks Base58Check with the 0x41 mainnet prefix.
func validTronAddress(address string) bool {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	n := new(big.Int)
	for _, r := range address {
		i := bytes.IndexRune([]byte(alphabet), r)
		if i < 0 {
			return false
		}
		n.Mul(n, big.NewInt(58)).Add(n, big.NewInt(int64(i)))
	}
	raw := n.Bytes()
	if len(raw) != 25 || raw[0] != 0x41 {
		return false
	}
	first := sha256.Sum256(raw[:21])
	second := sha256.Sum256(first[:])
	return bytes.Equal(second[:4], raw[21:])
}
