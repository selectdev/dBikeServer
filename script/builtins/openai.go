package builtins

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	tengo "github.com/d5/tengo/v2"

	"dbikeserver/config"
)

func newOpenAIClient() openai.Client {
	return openai.NewClient(option.WithAPIKey(config.OpenAIAPIKey))
}

func parseMessagesForOpenAI(arr *tengo.Array, fnName string) ([]openai.ChatCompletionMessageParamUnion, error) {
	msgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(arr.Value))
	for i, elem := range arr.Value {
		m, ok := elem.(*tengo.Map)
		if !ok {
			return nil, fmt.Errorf("%s: messages[%d] must be a map", fnName, i)
		}
		roleObj, _ := m.Value["role"].(*tengo.String)
		contentObj, _ := m.Value["content"].(*tengo.String)
		if roleObj == nil || contentObj == nil {
			return nil, fmt.Errorf("%s: messages[%d] must have string keys 'role' and 'content'", fnName, i)
		}
		switch roleObj.Value {
		case "system":
			msgs = append(msgs, openai.SystemMessage(contentObj.Value))
		case "user":
			msgs = append(msgs, openai.UserMessage(contentObj.Value))
		case "assistant":
			msgs = append(msgs, openai.AssistantMessage(contentObj.Value))
		default:
			return nil, fmt.Errorf("%s: messages[%d] unknown role %q", fnName, i, roleObj.Value)
		}
	}
	return msgs, nil
}

func openaiChatFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "openai_chat",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("openai_chat: expected (model, messages)")
			}
			model, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("openai_chat: model must be a string")
			}
			arr, ok := args[1].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("openai_chat: messages must be an array of maps")
			}
			msgs, err := parseMessagesForOpenAI(arr, "openai_chat")
			if err != nil {
				return tengo.UndefinedValue, err
			}
			client := newOpenAIClient()
			resp, err := client.Chat.Completions.New(context.Background(),
				openai.ChatCompletionNewParams{
					Model:    model.Value,
					Messages: msgs,
				},
			)
			if err != nil {
				return tengo.UndefinedValue, fmt.Errorf("openai_chat: %w", err)
			}
			return &tengo.String{Value: resp.Choices[0].Message.Content}, nil
		},
	}
}

func openaiChatExFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "openai_chat_ex",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("openai_chat_ex: expected (model, messages)")
			}
			model, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("openai_chat_ex: model must be a string")
			}
			arr, ok := args[1].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("openai_chat_ex: messages must be an array of maps")
			}
			msgs, err := parseMessagesForOpenAI(arr, "openai_chat_ex")
			if err != nil {
				return tengo.UndefinedValue, err
			}
			client := newOpenAIClient()
			resp, err := client.Chat.Completions.New(context.Background(),
				openai.ChatCompletionNewParams{
					Model:    model.Value,
					Messages: msgs,
				},
			)
			if err != nil {
				return tengo.UndefinedValue, fmt.Errorf("openai_chat_ex: %w", err)
			}
			ch := resp.Choices[0]
			return &tengo.Map{Value: map[string]tengo.Object{
				"content":           &tengo.String{Value: ch.Message.Content},
				"finish_reason":     &tengo.String{Value: string(ch.FinishReason)},
				"prompt_tokens":     &tengo.Int{Value: resp.Usage.PromptTokens},
				"completion_tokens": &tengo.Int{Value: resp.Usage.CompletionTokens},
				"total_tokens":      &tengo.Int{Value: resp.Usage.TotalTokens},
			}}, nil
		},
	}
}
