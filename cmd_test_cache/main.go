package main

import (
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

func main() {
	cc := anthropic.NewCacheControlEphemeralParam()
	fmt.Printf("CacheControl value: %+v\n", cc)
	fmt.Printf("IsOmitted: %v\n", param.IsOmitted(cc))

	tool := anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Properties: map[string]any{"city": map[string]any{"type": "string"}},
			Required:   []string{"city"},
		},
		"get_weather",
	)
	tool.OfTool.Description = anthropic.String("Get weather")
	tool.OfTool.CacheControl = anthropic.NewCacheControlEphemeralParam()

	data, _ := json.MarshalIndent(tool, "", "  ")
	fmt.Println("Tool JSON:")
	fmt.Println(string(data))

	tb := anthropic.TextBlockParam{
		Text:         "Hello",
		CacheControl: anthropic.NewCacheControlEphemeralParam(),
	}
	data2, _ := json.MarshalIndent(tb, "", "  ")
	fmt.Println("TextBlockParam JSON:")
	fmt.Println(string(data2))
}
