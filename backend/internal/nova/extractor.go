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

const extractionPrompt = `You are an expert software engineer. Extract factual knowledge statements that are EXPLICITLY present in the text below.

Extract only:
- Business rules (e.g. who can do what, approval thresholds, role restrictions)
- Domain constraints (e.g. numeric limits, time windows, valid state transitions)
- System behaviors (e.g. what happens on timeout, retry logic, error conditions)
- Architectural decisions (e.g. which service owns what, how traffic flows, technology choices)
- Data rules (e.g. where data is stored, schema details, retention policies)

SKIP — do not extract:
- HTTP status code assertions (e.g. "returns 200", "status is 400", "expected status for X is 200")
- Test setup/teardown boilerplate
- Generic CRUD confirmations with no domain meaning (e.g. "creating X returns 201")
- Import statements, variable declarations, and scaffolding with no business meaning

Strict rules:
- Only extract facts EXPLICITLY stated in the text — do not infer or invent
- Every number, name, and condition must come directly from the text
- Each fact must be self-contained: name the specific service, component, entity, state, or value involved so the fact is understandable without any surrounding context
- Prefer facts that explain WHY or WHAT the system enforces, not just HTTP response codes
- If the text contains no meaningful facts, return []
- Do not use prior knowledge from other files. Only use the text provided.

Output: a JSON array of objects with "fact" (complete English sentence) and "type".
Valid types: business_rule, architecture, data_rule, behavior, constraint, decision

Text to extract from:
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
