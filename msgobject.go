package bmwire

// Objects in bitmessage are things on the network that get
// propagated. This can include requests/responses for pubkeys,
// messages and broadcasts.

type ObjectType uint32

const (
	ObjectTypeGetPubKey ObjectType = 0
	ObjectTypePubKey    ObjectType = 1
	ObjectTypeMsg       ObjectType = 2
	ObjectTypeBroadcast ObjectType = 3
)

// Map of service flags back to their constant names for pretty printing.
var obStrings = map[ObjectType]string{
	ObjectTypeGetPubKey: "GETPUBKEY",
	ObjectTypePubKey:    "PUBKEY",
	ObjectTypeMsg:       "MSG",
	ObjectTypeBroadcast: "BROADCAST",
}
