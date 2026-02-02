package telegram

import "testing"

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCmd     string
		wantContent string
	}{
		{
			name:        "inbox command with content",
			input:       "/inbox Buy groceries",
			wantCmd:     "/inbox",
			wantContent: "Buy groceries",
		},
		{
			name:        "inbox command with long content",
			input:       "/inbox This is a very long message that should be captured in full",
			wantCmd:     "/inbox",
			wantContent: "This is a very long message that should be captured in full",
		},
		{
			name:        "status command",
			input:       "/status",
			wantCmd:     "/status",
			wantContent: "",
		},
		{
			name:        "unknown command",
			input:       "/help",
			wantCmd:     "",
			wantContent: "/help",
		},
		{
			name:        "plain text",
			input:       "hello world",
			wantCmd:     "",
			wantContent: "hello world",
		},
		{
			name:        "inbox without space is not a command",
			input:       "/inboxfoo",
			wantCmd:     "",
			wantContent: "/inboxfoo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, content := ParseCommand(tt.input)
			if cmd != tt.wantCmd {
				t.Errorf("ParseCommand(%q) command = %q, want %q", tt.input, cmd, tt.wantCmd)
			}
			if content != tt.wantContent {
				t.Errorf("ParseCommand(%q) content = %q, want %q", tt.input, content, tt.wantContent)
			}
		})
	}
}

func TestTruncateTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "short content unchanged",
			content: "Buy groceries",
			want:    "Buy groceries",
		},
		{
			name:    "exactly 20 chars unchanged",
			content: "12345678901234567890",
			want:    "12345678901234567890",
		},
		{
			name:    "over 20 chars truncated",
			content: "This is a very long message",
			want:    "This is a very long ...",
		},
		{
			name:    "21 chars truncated",
			content: "123456789012345678901",
			want:    "12345678901234567890...",
		},
		{
			name:    "empty string",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateTitle(tt.content)
			if got != tt.want {
				t.Errorf("TruncateTitle(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}
