package cache

import (
	"C0de1ndex/internal/types"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

var defaultManager = &cacheManager{
	cacheDir: ".c0de1ndex_cache",
}

type cacheManager struct {
	cacheDir string
}

// InitCache ensures the cache directory exists.
// InitCache 确保缓存目录存在。
func InitCache() error {
	return os.MkdirAll(defaultManager.cacheDir, 0755)
}

// GetContentHash calculates the SHA256 hash of a byte slice.
// GetContentHash 计算字节切片的 SHA256 哈希值。
func GetContentHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// GetCachedAnalysis attempts to retrieve a cached analysis for a given content hash.
// GetCachedAnalysis 尝试根据给定的内容哈希检索缓存的分析结果。
func GetCachedAnalysis(hash string) (*types.FileAnalysis, error) {
	cacheFile := filepath.Join(defaultManager.cacheDir, hash+".json")

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Cache miss
		}
		return nil, err // Other error
	}

	var analysis types.FileAnalysis
	if err := json.Unmarshal(data, &analysis); err != nil {
		return nil, err // Corrupted cache file
	}

	return &analysis, nil
}

// SaveAnalysisToCache saves a file analysis to the cache.
// SaveAnalysisToCache 将文件分析结果保存到缓存。
func SaveAnalysisToCache(hash string, analysis *types.FileAnalysis) error {
	cacheFile := filepath.Join(defaultManager.cacheDir, hash+".json")

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// SetCacheDir sets the cache directory for testing purposes.
func SetCacheDir(dir string) {
	defaultManager.cacheDir = dir
}

// GetCacheDir gets the current cache directory.
func GetCacheDir() string {
	return defaultManager.cacheDir
}