package hash

type Hasher interface {
	// Generate takes your parameter map or struct, sorts them, appends keys, hashes via SHA256, and returns uppercase hex.
	GenerateCheckMacVal(params map[string]string) (string, error)

	// Validate compares the incoming CheckMacValue against a newly generated one from the params.
	ValidateCheckMacVal(params map[string]string, incomingMAC string) (bool, error)
}
