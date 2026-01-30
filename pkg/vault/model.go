package vault

// CommonFrontmatter contains fields common to all notes
type CommonFrontmatter struct {
	Created  string   `yaml:"created"`
	Tags     []string `yaml:"tags,omitempty"`
	Priority string   `yaml:"priority,omitempty"` // low, medium, high, urgent
}

// InboxItem represents an item in the Inbox
type InboxItem struct {
	CommonFrontmatter `yaml:",inline"`
	Status            string `yaml:"status"` // inbox
	Type              string `yaml:"type"`   // task, idea, reference, project
}

// Project represents an active project
type Project struct {
	CommonFrontmatter `yaml:",inline"`
	Status            string `yaml:"status"` // planning, active, on-hold, completed, cancelled
	Type              string `yaml:"type"`   // project
	DueDate           string `yaml:"due_date,omitempty"`
	ReviewDate        string `yaml:"review_date,omitempty"`
}

// NextAction represents a task with a context
type NextAction struct {
	CommonFrontmatter `yaml:",inline"`
	Status            string `yaml:"status"`  // next, waiting, scheduled, delegated
	Context           string `yaml:"context"` // @calls, @computer, etc.
	Project           string `yaml:"project,omitempty"`
	DueDate           string `yaml:"due_date,omitempty"`
}

// WeeklyReview represents a weekly review note
type WeeklyReview struct {
	CommonFrontmatter `yaml:",inline"`
	Type              string `yaml:"type"` // weekly-review
	WeekOf            string `yaml:"week_of"`
}

// Note represents a parsed markdown note
type Note struct {
	Path        string
	Frontmatter interface{}
	Content     string // The markdown content after frontmatter
}
