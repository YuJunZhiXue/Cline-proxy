package kit

import (
	"os"
	"path/filepath"
)

// ResolveDataPath 数据文件路径解析：优先 data/ 子目录（可执行文件目录，其次工作目录），
// 兼容历史根目录存放；均不存在时默认写到 data/ 子目录并自动创建目录。
// go run 运行时编译产物在临时目录，此时应回退到工作目录（项目根）的 data/。
func ResolveDataPath(filename string) string {
	exeDir, pwd := "", ""
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	if wd, err := os.Getwd(); err == nil {
		pwd = wd
	}
	candidates := []string{}
	if exeDir != "" {
		candidates = append(candidates, filepath.Join(exeDir, "data", filename))
	}
	if pwd != "" {
		candidates = append(candidates, filepath.Join(pwd, "data", filename))
	}
	if exeDir != "" {
		candidates = append(candidates, filepath.Join(exeDir, filename))
	}
	if pwd != "" {
		candidates = append(candidates, filepath.Join(pwd, filename))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// 均不存在：默认写到可执行文件目录 data/（go run 场景回退工作目录 data/），
	// 账号数据统一收敛在 data/ 下，避免散落项目根目录。
	if exeDir != "" {
		os.MkdirAll(filepath.Join(exeDir, "data"), 0755)
		return filepath.Join(exeDir, "data", filename)
	}
	os.MkdirAll(filepath.Join(pwd, "data"), 0755)
	return filepath.Join(pwd, "data", filename)
}

// FileExists 判断文件是否存在。
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
