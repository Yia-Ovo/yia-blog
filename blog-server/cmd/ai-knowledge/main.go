// ai-knowledge 导出私有助手允许检索的资料副本，不包含运维配置、凭据和历史归档。
package main

import (
	infrarepo "blog-server/infra/repo"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	source := flag.String("source", "..", "仓库根目录")
	output := flag.String("output", "", "新建资料导出目录（须在仓库 .review-commit-prep 内）")
	flag.Parse()
	root, err := filepath.Abs(*source)
	if err != nil {
		fail(err)
	}
	target, err := filepath.Abs(*output)
	if err != nil {
		fail(err)
	}
	allowed := filepath.Join(root, ".review-commit-prep") + string(filepath.Separator)
	if *output == "" || !strings.HasPrefix(strings.ToLower(target), strings.ToLower(allowed)) {
		fail(fmt.Errorf("导出目录必须位于 %s", allowed))
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		fail(fmt.Errorf("导出目录须不存在，避免混入旧资料"))
	}
	paths, err := infrarepo.AIKnowledgeFiles(root)
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(target, 0700); err != nil {
		fail(err)
	}
	count := 0
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			fail(err)
		}
		if info.Size() > 2<<20 {
			continue
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			fail(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}
		destination := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			fail(err)
		}
		if err := os.WriteFile(destination, []byte(infrarepo.RedactAIKnowledge(string(data))), 0600); err != nil {
			fail(err)
		}
		if err := os.Chtimes(destination, info.ModTime(), info.ModTime()); err != nil {
			fail(err)
		}
		count++
	}
	fmt.Printf("已导出 %d 份资料：%s\n", count, target)
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
