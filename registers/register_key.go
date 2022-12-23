package registers

import (
	"encoding/hex"
	"github.com/onflow/flow-go/model/flow"
)

type RegisterKey struct {
	Owner string
	Key   string
}

func (key RegisterKey) IsSlab() bool {
	return len(key.Key) > 0 && key.Key[0] == '$'
}

func (key RegisterKey) ToReadable() RegisterKey {
	a := flow.BytesToAddress([]byte(key.Owner))
	var keyString string

	if key.IsSlab() {
		keyString = "$" + hex.EncodeToString([]byte(key.Key[1:]))
	} else {
		keyString = key.Key
	}

	return RegisterKey{
		Owner: a.Hex(),
		Key:   keyString,
	}
}

func (key RegisterKey) ToMangled() RegisterKey {
	a := flow.HexToAddress(key.Owner)
	keyString := key.Key
	if key.IsSlab() {
		decoded, err := hex.DecodeString(key.Key[1:])
		if err == nil {
			keyString = "$" + string(decoded)
		}
	}

	return RegisterKey{
		Owner: string(a.Bytes()),
		Key:   keyString,
	}
}

func (key RegisterKey) String() string {
	return "[" + key.ToReadable().Owner + "]: " + key.ToReadable().Key
}
