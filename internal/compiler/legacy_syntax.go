package compiler

// WP1/WP2 slice: recognized legacy syntax priority over UNKNOWN_FIELD.
//
// CA-SL §24.1 errata: after a safe YAML parse, a determinately recognized old
// Wait/Await control structure reports UNSUPPORTED_WAIT, and a known v0.2 legacy
// field (for example a Node `verify`) reports UNSUPPORTED_V02_SYNTAX. Both take
// priority over UNKNOWN_FIELD for the same structure. Removed v1alpha3-only
// author fields such as `operation.target` stay UNKNOWN_FIELD.

// legacyFieldClass classifies one non-CA-SL field name.
func legacyFieldClass(name string) string {
	switch name {
	case "wait", "await", "waitFor", "waitUntil", "waitPolicy", "awaitFor", "awaitUntil", "waitCondition", "waitTimeout":
		return "wait"
	case "verify", "steps", "localTools", "manifest":
		return "v02"
	}
	return ""
}
