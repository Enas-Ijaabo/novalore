package nova

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/enas/orglens/internal/pipeline"
)

const maxSynthesisFacts = 10

const synthesisPrompt = `You are a senior software engineer answering questions about a codebase.

You have been given a set of knowledge statements extracted directly from the source code and documentation.

Knowledge statements:
{facts}

Question: {question}

Rules:
- Base your answer strictly on the provided knowledge statements.
- Do not infer or add information not present in the statements.
- Every claim must include a source citation in brackets, e.g. [auth.go].
- If the statements do not contain enough information to answer, say so explicitly.`

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
		fmt.Fprintf(&sb, "%d. %s [%s]\n", i+1, f.Text, filepath.Base(f.Source))
	}

	prompt := strings.ReplaceAll(synthesisPrompt, "{facts}", sb.String())
	prompt = strings.ReplaceAll(prompt, "{question}", question)

	return c.Invoke(ctx, prompt)
}
