package artifact

// Generated Script Output Schema identity and path derivation, per the frozen
// Compiler Artifact Contract §2.

import (
	"fmt"
	"strings"
)

// ScriptOutputSchemaPath derives the default generated Schema path for a
// Procedure Script, e.g. skills/s/scripts/common/build_policy.py becomes
// skills/s/schemas/generated/common/build_policy.output.schema.json.
func ScriptOutputSchemaPath(skill, scriptPath string) (string, error) {
	relative, err := scriptRelativePath(scriptPath)
	if err != nil {
		return "", err
	}
	withoutSuffix := strings.TrimSuffix(relative, ".py")
	path := "skills/" + skill + "/schemas/generated/" + withoutSuffix + ".output.schema.json"
	if err := validateOutputPath(path); err != nil {
		return "", err
	}
	return path, nil
}

// ScriptOutputSchemaID derives the exact generated Schema $id:
// urn:connectionagent:skill:<encoded skill>:script-output:<encoded script path>.
func ScriptOutputSchemaID(skill, scriptPath string) (string, error) {
	if !validSkillName(skill) {
		return "", fmt.Errorf("invalid Skill name %q", skill)
	}
	relative, err := scriptRelativePath(scriptPath)
	if err != nil {
		return "", err
	}
	return "urn:connectionagent:skill:" + encodeRFC3986(skill) + ":script-output:" + encodeRFC3986(relative), nil
}

func scriptRelativePath(scriptPath string) (string, error) {
	if !strings.HasPrefix(scriptPath, "scripts/") || !strings.HasSuffix(scriptPath, ".py") {
		return "", fmt.Errorf("a Script path must start with scripts/ and end with .py: %q", scriptPath)
	}
	relative := strings.TrimPrefix(scriptPath, "scripts/")
	if relative == "" {
		return "", fmt.Errorf("a Script path must name a file under scripts/")
	}
	for _, segment := range strings.Split(relative, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("Script path is not normalized: %q", scriptPath)
		}
	}
	return relative, nil
}

// encodeRFC3986 percent-encodes every UTF-8 byte outside RFC 3986 unreserved
// characters with uppercase %HH. A slash is encoded as %2F.
func encodeRFC3986(value string) string {
	const hex = "0123456789ABCDEF"
	var out strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '.' || c == '_' || c == '~' {
			out.WriteByte(c)
			continue
		}
		out.WriteByte('%')
		out.WriteByte(hex[c>>4])
		out.WriteByte(hex[c&0x0f])
	}
	return out.String()
}
