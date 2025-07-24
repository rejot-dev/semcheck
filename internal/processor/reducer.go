package processor

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/rejot-dev/semcheck/internal/config"
	"github.com/rejot-dev/semcheck/internal/providers"
)

type ReducerQueryResponse struct {
	Query string
}

type Reducer struct {
	limit         int
	contextBefore int
	contextAfter  int
	client        providers.OllamaClient[ReducerQueryResponse]
}

func NewReducer(limit int, contextBefore int, contextAfter int, client providers.OllamaClient[ReducerQueryResponse]) (*Reducer, error) {
	m := &Reducer{
		limit:         limit,
		contextBefore: contextBefore,
		contextAfter:  contextAfter,
		client:        client,
	}
	return m, nil
}

func (r *Reducer) GetGrepQuery(ctx context.Context, rule config.Rule, specifically string, header string) (string, error) {
	data := PromptData{
		Header:          header,
		Specifically:    specifically,
		RuleName:        rule.Name,
		RuleDescription: rule.Description,
	}

	// Build user prompt
	userTmpl := template.Must(template.New("user").Parse(UserPromptTemplate))
	var userResult strings.Builder

	if err := userTmpl.Execute(&userResult, data); err != nil {
		return "", err
	}

	req := &providers.Request{
		UserPrompt:   userResult.String(),
		SystemPrompt: SystemPrompt,
	}

	fmt.Printf("User Prompt: %s\n", userResult.String())
	fmt.Printf("System Prompt: %s\n", SystemPrompt)

	resp, _, err := r.client.Complete(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Query, nil
}

func (r *Reducer) ExecuteGrep(content string, query string) (string, error) {
	re, err := regexp.Compile(query)
	if err != nil {
		return "", fmt.Errorf("Invalid regex pattern: %v", err)
	}

	lines := strings.Split(content, "\n")
	matchedLineIndices := make(map[int]bool)

	// Find all matching lines
	for i, line := range lines {
		if re.MatchString(line) {
			matchedLineIndices[i] = true
		}
	}

	if len(matchedLineIndices) == 0 {
		return "", fmt.Errorf("No matches found!")
	}

	// Add context lines around matches
	contextIndices := make(map[int]bool)
	for matchIdx := range matchedLineIndices {
		// Add lines before the match
		for i := matchIdx - r.contextBefore; i <= matchIdx+r.contextAfter; i++ {
			if i >= 0 && i < len(lines) {
				contextIndices[i] = true
			}
		}
	}

	// Collect lines in order
	var resultLines []string
	for i := range lines {
		if contextIndices[i] {
			resultLines = append(resultLines, lines[i])
		}
	}

	return strings.Join(resultLines, "\n"), nil
}

func (r *Reducer) ReduceRegex(ctx context.Context, rule config.Rule, contents string, specifically string) (string, error) {
	if len(contents) <= r.limit {
		fmt.Println("Content length is less than the limit, no reduction applied")
		return contents, nil
	}

	header := contents[:r.limit]

	// Query AI for content reduction queries
	query, err := r.GetGrepQuery(ctx, rule, specifically, header)
	if err != nil {
		return "", err
	}
	fmt.Println("QUERY:")
	fmt.Println(query)

	reduced, err := r.ExecuteGrep(contents, query)
	if err != nil {
		return "", err
	}
	fmt.Println("REDUCED:")
	fmt.Println(reduced)

	return reduced, nil
}

func (r *Reducer) Reduce(ctx context.Context, rule config.Rule, contents string, specifically string) (string, error) {
	if len(contents) <= r.limit {
		fmt.Println("Content length is less than the limit, no reduction applied")
		return contents, nil
	}

	fmt.Println("Specifically:", specifically)

	query_embedding, err := r.client.Embed(ctx, &providers.Request{
		SystemPrompt: specifically,
		UserPrompt:   specifically,
	})

	if err != nil {
		return "", err
	}

	// Step 1: Split contents into lines
	lines := strings.Split(contents, "\n")

	// Step 2: Embed each line
	type LineWithSimilarity struct {
		LineIndex  int
		Line       string
		Similarity float64
	}

	var linesSimilarity []LineWithSimilarity

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue // Skip empty lines
		}

		line_embedding, err := r.client.Embed(ctx, &providers.Request{
			SystemPrompt: "",
			UserPrompt:   line,
		})

		if err != nil {
			return "", err
		}

		// Step 3: Perform cosine similarity
		similarity := cosineSimilarity(query_embedding, line_embedding)
		linesSimilarity = append(linesSimilarity, LineWithSimilarity{
			LineIndex:  i,
			Line:       line,
			Similarity: similarity,
		})
	}

	// Sort by similarity in descending order
	sort.Slice(linesSimilarity, func(i, j int) bool {
		return linesSimilarity[i].Similarity > linesSimilarity[j].Similarity
	})

	for _, lineSim := range linesSimilarity {
		fmt.Println("Line:", lineSim.Line, "\nSimilarity:", lineSim.Similarity)
	}

	// Step 4: Return reduced context with top matches and context
	var selectedIndices map[int]bool = make(map[int]bool)

	// Take top matches until we reach the limit
	currentLength := 0
	for _, lineSim := range linesSimilarity {
		// Add context lines around the match
		for contextIdx := lineSim.LineIndex - r.contextBefore; contextIdx <= lineSim.LineIndex+r.contextAfter; contextIdx++ {
			if contextIdx >= 0 && contextIdx < len(lines) {
				if !selectedIndices[contextIdx] {
					selectedIndices[contextIdx] = true
					currentLength += len(lines[contextIdx]) + 1 // +1 for newline
				}
			}
		}

		if currentLength >= r.limit {
			break
		}
	}

	// Collect selected lines in original order
	var resultLines []string
	for i := range lines {
		if selectedIndices[i] {
			resultLines = append(resultLines, lines[i])
		}
	}

	return strings.Join(resultLines, "\n"), nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
