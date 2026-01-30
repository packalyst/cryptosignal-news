package cache

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
)

// GenerateCacheKey creates a cache key from prefix and params
func GenerateCacheKey(prefix string, params ...interface{}) string {
	if len(params) == 0 {
		return prefix
	}

	data, _ := json.Marshal(params)
	hash := md5.Sum(data)
	return prefix + ":" + hex.EncodeToString(hash[:])
}

// GetETag returns a hash of the data for ETag support
func GetETag(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	hash := md5.Sum(jsonData)
	return `"` + hex.EncodeToString(hash[:]) + `"`
}
