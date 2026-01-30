package ai

import "fmt"

// AnalyzeInboxPrompt returns a prompt to analyze an inbox item
func AnalyzeInboxPrompt(content string) string {
	return fmt.Sprintf(`
You are a GTD (Getting Things Done) assistant. Analyze the following input and extract structured information.

Input: "%s"

Determine:
1. Type: (task, idea, reference, project)
2. Title: A concise title
3. Priority: (low, medium, high, urgent)
4. Suggested Context: (@calls, @computer, @errands, @home, @office, @waiting) if applicable
5. Description: A brief summary

Output as JSON:
{
  "type": "...",
  "title": "...",
  "priority": "...",
  "context": "...",
  "description": "..."
}
`, content)
}

// GenerateReviewPrompt returns a prompt to generate a weekly review
func GenerateReviewPrompt(activeProjects []string, inboxCount int) string {
	projectsList := ""
	for _, p := range activeProjects {
		projectsList += fmt.Sprintf("- %s\n", p)
	}

	return fmt.Sprintf(`
You are a GTD assistant. Help me generate a Weekly Review.

Context:
- Inbox Items Remaining: %d
- Active Projects:
%s

Instructions:
1. Summarize the state of the projects.
2. Suggest 3 key priorities for next week based on the active projects.
3. Provide a brief reflection prompt.

Output as Markdown suitable for the "Reflections" and "Goals & Priorities" sections of the Weekly Review template.
`, inboxCount, projectsList)
}
