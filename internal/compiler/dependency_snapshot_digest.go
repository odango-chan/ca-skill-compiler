package compiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
)

const dependencySnapshotDigestDomain = "CA-SKILL-DEPENDENCY-SNAPSHOT\x00v1\x00"

// ComputeDependencySnapshotDigest computes the frozen v1alpha4 binary
// DependencySnapshot digest. It hashes original file bytes and normalized
// snapshot-relative paths; dependencyId and input file order are excluded.
func ComputeDependencySnapshotDigest(files []File) (string, error) {
	if len(files) == 0 || len(files) > maxFiles {
		return "", fmt.Errorf("snapshot must contain 1 to %d files", maxFiles)
	}
	ordered := append([]File(nil), files...)
	seen := make(map[string]bool, len(ordered))
	total := 0
	for _, file := range ordered {
		if !validRelativeSnapshotPath(file.Path) {
			return "", fmt.Errorf("invalid snapshot-relative path %q", file.Path)
		}
		if seen[file.Path] {
			return "", fmt.Errorf("duplicate snapshot path %q", file.Path)
		}
		seen[file.Path] = true
		if len(file.Bytes) > maxFileSize {
			return "", fmt.Errorf("file %q exceeds %d bytes", file.Path, maxFileSize)
		}
		total += len(file.Bytes)
		if total > maxTotal {
			return "", fmt.Errorf("snapshot exceeds %d raw bytes", maxTotal)
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		return bytes.Compare([]byte(ordered[i].Path), []byte(ordered[j].Path)) < 0
	})

	hash := sha256.New()
	_, _ = hash.Write([]byte(dependencySnapshotDigestDomain))
	var u32 [4]byte
	var u64 [8]byte
	binary.BigEndian.PutUint32(u32[:], uint32(len(ordered)))
	_, _ = hash.Write(u32[:])
	for _, file := range ordered {
		pathBytes := []byte(file.Path)
		binary.BigEndian.PutUint32(u32[:], uint32(len(pathBytes)))
		_, _ = hash.Write(u32[:])
		_, _ = hash.Write(pathBytes)
		binary.BigEndian.PutUint64(u64[:], uint64(len(file.Bytes)))
		_, _ = hash.Write(u64[:])
		_, _ = hash.Write(file.Bytes)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
