package compiler

// VerifyBuildResultJSON decodes a BuildResult wire value and verifies its artifact
// tree integrity. It returns nil for a failed/non-success result and for an empty
// artifact tree, matching VerifyBuildResult.

import "encoding/json"

func VerifyBuildResultJSON(data []byte) error {
	var result BuildResult
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	return VerifyBuildResult(result)
}
