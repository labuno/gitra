package cli

import "errors"

// errUsage marks invalid CLI input: unknown commands/flags, missing required
// flags, or malformed arguments (baseline §35: exit code 2).
var errUsage = errors.New("invalid usage")
