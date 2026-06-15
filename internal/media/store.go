package media

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/qzone-memory/config"
)

// baseDir 返回媒体落盘根目录。
func baseDir() string {
	if config.GlobalConfig != nil && config.GlobalConfig.Media.Dir != "" {
		return config.GlobalConfig.Media.Dir
	}
	return "./data/media"
}

// relPathFor 计算资源的相对落点：<qq>/<category>/<hash[:2]>/<hash><ext>。
func relPathFor(userQQ, category, hash, ext string) string {
	if category == "" {
		category = "misc"
	}
	shard := hash
	if len(hash) >= 2 {
		shard = hash[:2]
	}
	return filepath.Join(userQQ, category, shard, hash+ext)
}

// absPath 把相对路径解析为根目录下的绝对路径，并防止路径穿越。
func absPath(rel string) (string, bool) {
	base, err := filepath.Abs(baseDir())
	if err != nil {
		return "", false
	}
	clean := filepath.Clean(filepath.Join(base, rel))
	if clean != base && !strings.HasPrefix(clean, base+string(os.PathSeparator)) {
		return "", false
	}
	return clean, true
}

// writeFile 原子写入：先写临时文件再 rename，避免出现半截文件。
func writeFile(rel string, data []byte) error {
	abs, ok := absPath(rel)
	if !ok {
		return errInvalidPath
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	tmp := abs + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, abs)
}

// fileExists 判断相对路径对应的文件是否存在。
func fileExists(rel string) bool {
	abs, ok := absPath(rel)
	if !ok {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && !info.IsDir()
}

// RemoveUserMedia 删除某 QQ 的全部本地媒体文件，用于「彻底删除我的数据」。
func RemoveUserMedia(userQQ string) error {
	abs, ok := absPath(userQQ)
	if !ok {
		return errInvalidPath
	}
	return os.RemoveAll(abs)
}
