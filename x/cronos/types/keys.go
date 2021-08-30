package types

const (
	// ModuleName defines the module name
	ModuleName = "cronos"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_cronos"

	// this line is used by starport scaffolding # ibc/keys/name
)

// prefix bytes for the cronos persistent store
const (
	prefixDenomToContract = iota + 1
)

// KVStore key prefixes
var (
	KeyPrefixDenomToContract = []byte{prefixDenomToContract}
)

// this line is used by starport scaffolding # ibc/keys/port

// DenomToContractKey defines the store key for denom to contract mapping
func DenomToContractKey(denom string) []byte {
	return append(KeyPrefixDenomToContract, denom...)
}
