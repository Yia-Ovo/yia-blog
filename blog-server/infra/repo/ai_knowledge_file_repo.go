package repo

import (
	"blog-server/domain/do"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// AIKnowledgeRoots 同时用于本机检索和独立资料导出；运维、简历和历史归档不在范围内。
var AIKnowledgeRoots = []string{".private/01-项目文档", ".private/02-面试资料", ".private/03-学习笔记", ".private/07-算法刷题", "blog-server/api", "blog-server/app", "blog-server/domain", "blog-server/infra", "blog-server/common", "blog-server/middleware", "blog-server/ioc", "blog-web/src"}

var aiSecretPattern = regexp.MustCompile(`(?i)(sk-[a-z0-9_-]{12,}|(?:api[_-]?key|password|secret|authorization)\s*[:=]\s*["']?[a-z0-9_/-]{16,})`)

type AIKnowledgeFileRepo struct {
	root      string
	mu        sync.Mutex
	loaded    time.Time
	documents int
	chunks    []do.AISource
}

func NewAIKnowledgeFileRepo(root string) *AIKnowledgeFileRepo {
	return &AIKnowledgeFileRepo{root: root}
}

// AIKnowledgeFiles 固定资料根和扩展名，不接受前端提交磁盘路径，也不跟随符号链接。
func AIKnowledgeFiles(root string) ([]string, error) {
	var paths []string
	for _, directory := range AIKnowledgeRoots {
		full := filepath.Join(root, filepath.FromSlash(directory))
		info, err := os.Lstat(full)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		err = filepath.WalkDir(full, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if entry.IsDir() {
				if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules" {
					return fs.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".md" && ext != ".go" && ext != ".ts" && ext != ".tsx" {
				return nil
			}
			if len(paths) >= 2000 {
				return errors.New("too many AI knowledge files")
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return paths, nil
}

// RedactAIKnowledge 对资料副本做凭据脱敏，不改写原笔记和原代码。
func RedactAIKnowledge(text string) string {
	return aiSecretPattern.ReplaceAllString(text, "[凭据已隐藏]")
}

func (r *AIKnowledgeFileRepo) refresh(ctx context.Context) error {
	if r.root == "" {
		return errors.New("AI knowledge directory is not configured")
	}
	if time.Since(r.loaded) < time.Minute {
		return nil
	}
	paths, err := AIKnowledgeFiles(r.root)
	if err != nil {
		return err
	}
	chunks := []do.AISource{}
	total := 0
	documents := 0
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Size() > 2<<20 {
			continue
		}
		total += int(info.Size())
		if total > 32<<20 {
			return errors.New("AI knowledge exceeds size limit")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(r.root, path)
		if err != nil {
			return err
		}
		lines := strings.Split(RedactAIKnowledge(string(data)), "\n")
		for start := 0; start < len(lines); {
			end := start
			count := 0
			for end < len(lines) && end-start < 70 {
				size := len([]rune(lines[end]))
				if count+size > 2000 && end > start {
					break
				}
				count += size
				end++
			}
			excerpt := []rune(strings.Join(lines[start:end], "\n"))
			if len(excerpt) > 2200 {
				excerpt = excerpt[:2200]
			}
			chunks = append(chunks, do.AISource{Path: filepath.ToSlash(relative), Line: start + 1, UpdatedAt: info.ModTime(), Excerpt: string(excerpt)})
			start = end
		}
		documents++
	}
	r.chunks = chunks
	r.documents = documents
	r.loaded = time.Now()
	return nil
}

func (r *AIKnowledgeFileRepo) Info(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.refresh(ctx); err != nil {
		return 0, err
	}
	return r.documents, nil
}

func aiTerms(query string) []string {
	query = strings.ToLower(query)
	terms := map[string]bool{}
	for _, word := range strings.FieldsFunc(query, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' }) {
		letters := []rune(word)
		if len(letters) >= 2 {
			terms[word] = true
		}
		for i := 0; i+1 < len(letters); i++ {
			if letters[i] > 127 && letters[i+1] > 127 {
				terms[string(letters[i:i+2])] = true
			}
		}
	}
	result := []string{}
	for term := range terms {
		result = append(result, term)
	}
	return result
}

func (r *AIKnowledgeFileRepo) Search(ctx context.Context, query string) ([]do.AISource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.refresh(ctx); err != nil {
		return nil, err
	}
	terms := aiTerms(query)
	type match struct {
		source do.AISource
		score  int
	}
	matches := []match{}
	for _, chunk := range r.chunks {
		text := strings.ToLower(chunk.Excerpt)
		path := strings.ToLower(chunk.Path)
		score := 0
		for _, term := range terms {
			if strings.Contains(text, term) {
				score++
			}
			if strings.Contains(path, term) {
				score += 3
			}
		}
		if chunk.Path == ".private/02-面试资料/README.md" && chunk.Line == 1 {
			score += 3
		}
		if score > 0 {
			matches = append(matches, match{chunk, score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
	result := []do.AISource{}
	perFile := map[string]int{}
	// 学习入口记录当前进度，优先纳入，避免新对话重新从第一题开始。
	for _, chunk := range r.chunks { if chunk.Path == ".private/02-面试资料/README.md" && chunk.Line == 1 { result = append(result, chunk); perFile[chunk.Path]++; break } }
	for _, m := range matches {
		if m.source.Path == ".private/02-面试资料/README.md" && m.source.Line == 1 && perFile[m.source.Path] > 0 { continue }
		if perFile[m.source.Path] >= 2 {
			continue
		}
		result = append(result, m.source)
		perFile[m.source.Path]++
		if len(result) == 6 {
			break
		}
	}
	return result, nil
}
