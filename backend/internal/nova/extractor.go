package nova

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/enas/orglens/internal/pipeline"
)

var validTypes = map[string]bool{
	"business_rule": true,
	"architecture":  true,
	"data_rule":     true,
	"behavior":      true,
	"constraint":    true,
	"decision":      true,
}

const extractionPrompt = `You are an expert software engineer analyzing source code and documentation.

Your job is to extract hidden knowledge statements — business rules, domain constraints,
system behaviors, architectural decisions — that are encoded in the text below.

Focus on knowledge that would be hard to discover without reading the code carefully:
- Business rules: "Free tier users are limited to 3 projects"
- Domain constraints: "Orders require a minimum charge of $10"
- System behaviors: "JWT tokens expire after 24 hours"
- Architectural decisions: "All external traffic routes through the API Gateway"
- Data rules: "Revoked tokens are stored in the revoked_tokens table"
- Retry/failure policies: "Payments are retried up to 3 times before failing"

Rules:
- Write each statement as a clear, complete English sentence
- Be specific — include numbers, names, and conditions when present
- Do not invent facts that are not explicitly in the text
- Skip trivial facts (imports, variable names, boilerplate)

Output format:
[
  {
    "fact": "JWT tokens expire after 24 hours",
    "type": "business_rule"
  },
  {
    "fact": "PaymentService uses Stripe for payment processing",
    "type": "architecture"
  }
]

Valid types: business_rule, architecture, data_rule, behavior, constraint, decision

Now extract from this text:
{text}

Return ONLY a valid JSON array. No explanation. No markdown.`

func (c *Client) ExtractFacts(ctx context.Context, text, source string) ([]pipeline.Fact, error) {
	prompt := strings.ReplaceAll(extractionPrompt, "{text}", text)

	raw, err := c.Invoke(ctx, prompt)
	if err != nil {
		return nil, err
	}

	raw = extractJSON(raw)
	if raw == "" {
		log.Printf("no JSON array found in model response\nraw: %s", raw)
		return []pipeline.Fact{}, nil
	}

	var items []struct {
		Fact string `json:"fact"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, fmt.Errorf("parse facts json: %w\nraw: %s", err, raw)
	}

	seen := map[string]bool{}
	facts := make([]pipeline.Fact, 0, len(items))
	for _, item := range items {
		item.Fact = strings.TrimSpace(item.Fact)
		item.Type = strings.TrimSpace(item.Type)

		if len(item.Fact) < 10 || len(item.Fact) > 200 {
			continue
		}
		if seen[item.Fact] {
			continue
		}
		seen[item.Fact] = true

		factType := item.Type
		if !validTypes[factType] {
			factType = "architecture"
		}

		facts = append(facts, pipeline.Fact{
			Text:   item.Fact,
			Type:   factType,
			Source: source,
		})
	}
	return facts, nil
}

// extractJSON finds the outermost JSON array in the model response,
// handling markdown code fences without panicking.
func extractJSON(raw string) string {
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start == -1 || end == -1 || end < start {
		return ""
	}
	return strings.TrimSpace(raw[start : end+1])
}
