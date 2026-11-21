package market

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"strings"
)

type Audit struct {
	Level    string   `json:"level"`
	Findings []string `json:"findings"`
	Files    []string `json:"files"`
}

var riskyCommands = []struct {
	Pattern string
	Label   string
}{
	{"curl", "包含 curl 网络请求"},
	{"wget", "包含 wget 网络请求"},
	{"| sh", "包含管道执行 shell"},
	{"| bash", "包含管道执行 bash"},
	{"rm -rf", "包含递归删除命令"},
	{"powershell -enc", "包含编码 PowerShell"},
	{"base64 -d", "包含 base64 解码"},
	{"invoke-expression", "包含 PowerShell 动态执行"},
}

func AuditArchive(archivePath string) (Audit, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return Audit{}, fmt.Errorf("打开安装包: %w", err)
	}
	defer reader.Close()
	audit := Audit{Level: "low", Findings: []string{}, Files: []string{}}
	ri := 0
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name)
		switch path.Ext(name) {
		case ".sh", ".ps1", ".bat", ".cmd", ".py", ".js", ".ts":
			audit.Files = append(audit.Files, entry.Name)
			if audit.Level == "low" {
				audit.Level = "medium"
			}
			audit.Findings = append(audit.Findings, fmt.Sprintf("包含脚本文件：%s", entry.Name))
		}
		if entry.UncompressedSize64 > 1<<20 {
			continue
		}
		file, err := entry.Open()
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(io.LimitReader(file, 1<<20))
		file.Close()
		text := strings.ToLower(string(data))
		for _, risky := range riskyCommands {
			if strings.Contains(text, risky.Pattern) {
				audit.Level = "high"
				audit.Findings = append(audit.Findings, fmt.Sprintf("%s：%s", entry.Name, risky.Label))
			}
		}
		ri++
		if ri > 200 {
			break
		}
	}
	if len(audit.Findings) > 20 {
		audit.Findings = audit.Findings[:20]
	}
	return audit, nil
}
