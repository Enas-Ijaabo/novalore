package nova

import (
	"context"
	"fmt"
	"strings"

	"github.com/enas/orglens/internal/pipeline"
)

const maxSynthesisFacts = 10

const synthesisPrompt = `You are a senior software engineer answering questions about a codebase.

You have been given knowledge statements extracted from the source code and documentation.

Knowledge statements:
{facts}

Question: {question}

Answer directly and concisely. Synthesize the relevant facts into a clear explanation — do not just list them. Skip implementation trivia (function names, status codes) unless they directly answer the question. Cite sources in brackets only for non-obvious claims, e.g. [order.go]. If the statements do not contain enough information, say so.`

// Synthesize generates a natural language answer from a question and retrieved facts.
func (c *Client) Synthesize(ctx context.Context, question string, facts []pipeline.Fact) (string, error) {
	if len(facts) == 0 {
		return "No relevant knowledge statements were found in the codebase for this question.", nil
	}
	if len(facts) > maxSynthesisFacts {
		facts = facts[:maxSynthesisFacts]
	}

	var sb strings.Builder
	for i, f := range facts {
		src := f.Source
	if after, ok := strings.CutPrefix(src, "repos/"); ok {
		src = after
	} else if after, ok := strings.CutPrefix(src, "docs/"); ok {
		src = after
	}
	fmt.Fprintf(&sb, "%d. %s [%s]\n", i+1, f.Text, src)
	}

	prompt := strings.ReplaceAll(synthesisPrompt, "{facts}", sb.String())
	prompt = strings.ReplaceAll(prompt, "{question}", question)

	return c.Invoke(ctx, prompt)
}
