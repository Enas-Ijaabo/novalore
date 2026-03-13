package nova

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

const embedModelID = "amazon.nova-embed-v1:0"

// Embed returns a 1024-dimensional vector for the given text.
func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	body, _ := json.Marshal(map[string]any{
		"inputText": text,
	})

	out, err := c.bedrock.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(embedModelID),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	var resp struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(out.Body, &resp); err != nil {
		return nil, fmt.Errorf("embed parse: %w", err)
	}
	if len(resp.Embedding) == 0 {
		return nil, fmt.Errorf("embed: empty vector returned")
	}
	return resp.Embedding, nil
}
