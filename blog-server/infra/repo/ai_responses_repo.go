package repo

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// AIResponsesRepo 使用服务器专属配置；拒绝重定向，避免转发 Authorization 到其他站点。
type AIResponsesRepo struct {
	baseURL, key, model string
	client              *http.Client
}

func NewAIResponsesRepo(baseURL, key, model string) *AIResponsesRepo {
	return &AIResponsesRepo{baseURL: strings.TrimRight(baseURL, "/"), key: key, model: model, client: &http.Client{Timeout: 95 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (r *AIResponsesRepo) Reply(ctx context.Context, instructions string, messages []do.AIModelMessage) (string, error) {
	input := []map[string]any{}
	for _, m := range messages {
		contentType := "input_text"
		if m.Role == "assistant" {
			contentType = "output_text"
		}
		input = append(input, map[string]any{"role": m.Role, "content": []map[string]string{{"type": contentType, "text": m.Content}}})
	}
	cacheHash := sha256.Sum256([]byte(r.key))
	cacheHash[6] = (cacheHash[6] & 0x0f) | 0x40
	cacheHash[8] = (cacheHash[8] & 0x3f) | 0x80
	cacheKey := fmt.Sprintf("%x-%x-%x-%x-%x", cacheHash[:4], cacheHash[4:6], cacheHash[6:8], cacheHash[8:10], cacheHash[10:16])
	// AnyRouter 的 Codex 通道拒绝 max_output_tokens；以请求额度、上下文边界和流式长度/时间上限控制使用。
	body, err := json.Marshal(map[string]any{"model": r.model, "instructions": instructions, "input": input, "stream": true, "store": false, "prompt_cache_key": cacheKey})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return "", errors.New("invalid AI endpoint")
	}
	request.Header.Set("Authorization", "Bearer "+r.key)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	response, err := r.client.Do(request)
	if err != nil {
		return "", errors.New("AI provider connection failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI provider returned HTTP %d", response.StatusCode)
	}
	if !strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
		data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
		if err != nil {
			return "", errors.New("AI response read failed")
		}
		return aiResponseText(data)
	}
	scanner := bufio.NewScanner(io.LimitReader(response.Body, 4<<20))
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	var output strings.Builder
	completed := false
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var event struct {
			Type     string          `json:"type"`
			Delta    string          `json:"delta"`
			Response json.RawMessage `json:"response"`
		}
		if json.Unmarshal([]byte(data), &event) != nil {
			return "", errors.New("invalid AI stream")
		}
		switch event.Type {
		case "response.output_text.delta":
			output.WriteString(event.Delta)
			if utf8.RuneCountInString(output.String()) > consts.AIAnswerMaxRunes {
				return "", errors.New("AI answer exceeds length limit")
			}
		case "response.completed":
			text, parseErr := aiResponseText(event.Response)
			if parseErr != nil {
				return "", parseErr
			}
			if output.Len() == 0 {
				output.WriteString(text)
			}
			completed = true
		case "response.failed", "response.incomplete", "error":
			return "", errors.New("AI generation did not complete")
		}
	}
	if scanner.Err() != nil || !completed || strings.TrimSpace(output.String()) == "" {
		return "", errors.New("AI stream interrupted")
	}
	return output.String(), nil
}

func aiResponseText(data []byte) (string, error) {
	var result struct {
		Status string `json:"status"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if json.Unmarshal(data, &result) != nil || result.Status != "completed" {
		return "", errors.New("AI response did not complete")
	}
	var output strings.Builder
	for _, item := range result.Output {
		if item.Type == "message" {
			for _, part := range item.Content {
				if part.Type == "output_text" {
					output.WriteString(part.Text)
				}
			}
		}
	}
	if strings.TrimSpace(output.String()) == "" {
		return "", errors.New("AI provider returned no answer")
	}
	if utf8.RuneCountInString(output.String()) > consts.AIAnswerMaxRunes {
		return "", errors.New("AI answer exceeds length limit")
	}
	return output.String(), nil
}
