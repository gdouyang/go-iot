package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	logs "go-iot/pkg/logger"
)

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string          `json:"content"`
			ReasoningContent string          `json:"reasoning_content"`
			Reasoning        json.RawMessage `json:"reasoning"`
			Thinking         string          `json:"thinking"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				Id       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		Message CompletionsMessage `json:"message"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

func (c *Client) ChatStream(ctx context.Context, baseURL, apiKey string, req CompletionsRequest, onDelta func(StreamDelta)) (*CompletionsResponse, error) {
	req.Stream = true
	if req.StreamOptions == nil {
		req.StreamOptions = &StreamOptions{IncludeUsage: true}
	}
	resp, raw, err := c.do(ctx, baseURL, apiKey, req)
	if err != nil {
		if ctx.Err() == nil {
			logs.Errorf("llm stream request error: %v", err)
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		logs.Errorf("llm http %d: %s", resp.StatusCode, string(body))
		return nil, &HTTPError{Status: resp.StatusCode, Body: string(body)}
	}
	ctype := resp.Header.Get("Content-Type")
	if !strings.Contains(ctype, "text/event-stream") && !strings.Contains(ctype, "text/plain") {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var out CompletionsResponse
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, err
		}
		if len(out.Choices) > 0 {
			out.Choices[0].Message.FillReasoning()
			emitStreamDelta(onDelta, StreamDelta{
				Reasoning: out.Choices[0].Message.Reasoning,
				Content:   out.Choices[0].Message.Content,
			})
		}
		return &out, nil
	}
	_ = raw
	return parseCompletionsSSE(resp.Body, onDelta)
}

func emitStreamDelta(onDelta func(StreamDelta), d StreamDelta) {
	if onDelta == nil {
		return
	}
	if d.Reasoning != "" {
		onDelta(StreamDelta{Reasoning: d.Reasoning})
	}
	if d.Content != "" {
		onDelta(StreamDelta{Content: d.Content})
	}
}

func (c *Client) do(ctx context.Context, baseURL, apiKey string, req CompletionsRequest) (*http.Response, []byte, error) {
	endpoint, err := JoinCompletionsURL(baseURL)
	if err != nil {
		return nil, nil, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	if req.Stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}
	cli := c.HTTP
	if cli == nil {
		cli = New().HTTP
	}
	resp, err := cli.Do(httpReq)
	return resp, body, err
}

func parseCompletionsSSE(r io.Reader, onDelta func(StreamDelta)) (*CompletionsResponse, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	var content strings.Builder
	var reasoning strings.Builder
	type acc struct {
		id, typ, name, args string
	}
	calls := map[int]*acc{}
	var maxIdx int
	var usage Usage
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var ch streamChunk
		if err := json.Unmarshal([]byte(payload), &ch); err != nil {
			continue
		}
		if ch.Usage.PromptTokens > 0 || ch.Usage.CompletionTokens > 0 || ch.Usage.ReasoningTokenCount() > 0 {
			usage = ch.Usage
		}
		if len(ch.Choices) == 0 {
			continue
		}
		delta := ch.Choices[0].Delta
		if piece := reasoningDelta(delta.ReasoningContent, delta.Reasoning, delta.Thinking); piece != "" {
			reasoning.WriteString(piece)
			if onDelta != nil {
				onDelta(StreamDelta{Reasoning: piece})
			}
		}
		if delta.Content != "" {
			content.WriteString(delta.Content)
			if onDelta != nil {
				onDelta(StreamDelta{Content: delta.Content})
			}
		}
		for _, tc := range delta.ToolCalls {
			a := calls[tc.Index]
			if a == nil {
				a = &acc{}
				calls[tc.Index] = a
			}
			if tc.Index > maxIdx {
				maxIdx = tc.Index
			}
			if tc.Id != "" {
				a.id = tc.Id
			}
			if tc.Type != "" {
				a.typ = tc.Type
			}
			if tc.Function.Name != "" {
				a.name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				a.args += tc.Function.Arguments
			}
		}
		if ch.Choices[0].Message.Content != "" && content.Len() == 0 {
			content.WriteString(ch.Choices[0].Message.Content)
			if onDelta != nil {
				onDelta(StreamDelta{Content: ch.Choices[0].Message.Content})
			}
		}
	}
	if err := sc.Err(); err != nil {
		logs.Errorf("llm stream parse error: %v", err)
		return nil, err
	}
	msg := CompletionsMessage{Role: "assistant", Content: content.String(), Reasoning: reasoning.String()}
	if len(calls) > 0 {
		for i := 0; i <= maxIdx; i++ {
			a := calls[i]
			if a == nil {
				continue
			}
			var tc ToolCall
			tc.Id = a.id
			tc.Type = a.typ
			if tc.Type == "" {
				tc.Type = "function"
			}
			tc.Function.Name = a.name
			tc.Function.Arguments = a.args
			msg.ToolCalls = append(msg.ToolCalls, tc)
		}
	}
	out := &CompletionsResponse{}
	out.Choices = []struct {
		Message CompletionsMessage `json:"message"`
	}{{Message: msg}}
	out.Usage = usage
	return out, nil
}
