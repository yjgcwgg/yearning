package fetch

import (
	"Yearning-go/src/lib/factory"
	"Yearning-go/src/model"
	"context"
	"github.com/sashabaranov/go-openai"
	"net/http"
	"net/url"
	"strings"
)

type AIAssistant struct {
	cc  *openai.Client
	req openai.ChatCompletionRequest
}

func replace(sql, kind string, tables []string) string {
	pp := model.GloAI.AdvisorPrompt
	if kind == "text2sql" {
		pp = model.GloAI.SQLGenPrompt
	}
	p := strings.ReplaceAll(pp, "{{tables_info}}", strings.Join(tables, "\n"))
	p = strings.ReplaceAll(p, "{{sql}}", sql)
	p = strings.ReplaceAll(p, "{{lang}}", model.C.General.Lang)
	return p
}

func NewAIAgent() (*AIAssistant, error) {
	ai := new(AIAssistant)
	ai.req = openai.ChatCompletionRequest{
		Model:            model.GloAI.Model,
		MaxTokens:        model.GloAI.MaxTokens,
		Temperature:      model.GloAI.Temperature,
		PresencePenalty:  model.GloAI.PresencePenalty,
		FrequencyPenalty: model.GloAI.FrequencyPenalty,
		TopP:             model.GloAI.TopP,
	}
	config := openai.DefaultConfig(model.GloAI.APIKey)
	config.BaseURL = model.GloAI.BaseUrl
	if model.GloAI.ProxyURL != "" {
		proxyUrl, err := url.Parse(model.GloAI.ProxyURL)
		if err != nil {
			return nil, err
		}
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		}
		config.HTTPClient = &http.Client{
			Transport: transport,
		}
	}
	ai.cc = openai.NewClientWithConfig(config)
	return ai, nil
}

func (ai *AIAssistant) Messages(messages []openai.ChatCompletionMessage) *AIAssistant {
	ai.req.Messages = messages
	return ai
}

func (ai *AIAssistant) BuildSQLAdvise(prompt *advisorFrom, tables []string, kind string) (string, error) {
	sql, err := factory.GetFingerprint(prompt.SQL)
	if err != nil {
		return "", err
	}
	ai.req.Messages = []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: replace(sql, kind, tables),
		},
	}
	resp, e := ai.cc.CreateChatCompletion(context.Background(), ai.req)
	if e != nil {
		return "", e
	}
	return resp.Choices[0].Message.Content, nil
}

func (ai *AIAssistant) StreamChatCompletion() (*openai.ChatCompletionStream, error) {
	ai.req.Stream = true
	stream, err := ai.cc.CreateChatCompletionStream(context.Background(), ai.req)
	if err != nil {
		return nil, err
	}
	return stream, nil
}
