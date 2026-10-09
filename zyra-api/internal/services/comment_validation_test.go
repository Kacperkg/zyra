package services

import (
	"fmt"
	"strings"
	"testing"
	"zyra-api/internal/models"
)

func TestRichCommentAllowlistAndLimits(t *testing.T) {
	valid := func() CommentInput {
		return CommentInput{SchemaVersion: 1, Content: &models.CommentContent{Blocks: []models.CommentBlock{{Type: "paragraph", Children: []models.CommentInline{{Type: "text", Text: "Investigated", Bold: true, Italic: true, Underline: true, Strike: true, Size: 16, Color: "#23a45F", Href: "https://example.test/runbook"}}}}}}
	}
	if _, _, err := validateComment(valid()); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*CommentInput)
	}{
		{"wrong version", func(in *CommentInput) { in.SchemaVersion = 2 }},
		{"HTML block", func(in *CommentInput) { in.Content.Blocks[0].Type = "html" }},
		{"image inline", func(in *CommentInput) { in.Content.Blocks[0].Children[0].Type = "image" }},
		{"invalid alignment", func(in *CommentInput) { in.Content.Blocks[0].Align = "justify" }},
		{"javascript link", func(in *CommentInput) { in.Content.Blocks[0].Children[0].Href = "javascript:alert(1)" }},
		{"data link", func(in *CommentInput) { in.Content.Blocks[0].Children[0].Href = "data:text/html,test" }},
		{"credential link", func(in *CommentInput) { in.Content.Blocks[0].Children[0].Href = "https://user:password@example.test" }},
		{"protocol-relative link", func(in *CommentInput) { in.Content.Blocks[0].Children[0].Href = "//example.test" }},
		{"arbitrary colour", func(in *CommentInput) { in.Content.Blocks[0].Children[0].Color = "red;position:absolute" }},
		{"arbitrary size", func(in *CommentInput) { in.Content.Blocks[0].Children[0].Size = 999 }},
		{"text mention forgery", func(in *CommentInput) { in.Content.Blocks[0].Children[0].UserID = "someone" }},
		{"mention label forgery", func(in *CommentInput) {
			in.Content.Blocks[0].Children = []models.CommentInline{{Type: "mention", UserID: "someone", Text: "Wrong Name"}}
		}},
		{"empty", func(in *CommentInput) {
			in.Content.Blocks[0].Children = []models.CommentInline{{Type: "text", Text: " \n\t"}}
		}},
		{"text too long", func(in *CommentInput) { in.Content.Blocks[0].Children[0].Text = strings.Repeat("x", 20001) }},
		{"too many blocks", func(in *CommentInput) {
			block := in.Content.Blocks[0]
			in.Content.Blocks = make([]models.CommentBlock, 101)
			for i := range in.Content.Blocks {
				in.Content.Blocks[i] = block
			}
		}},
		{"too many nodes", func(in *CommentInput) {
			in.Content.Blocks[0].Children = make([]models.CommentInline, 1001)
			for i := range in.Content.Blocks[0].Children {
				in.Content.Blocks[0].Children[i] = models.CommentInline{Type: "text", Text: "x"}
			}
		}},
		{"too many recipients", func(in *CommentInput) {
			in.Content.Blocks[0].Children = nil
			for i := 0; i < 21; i++ {
				in.Content.Blocks[0].Children = append(in.Content.Blocks[0].Children, models.CommentInline{Type: "mention", UserID: fmt.Sprint(i)})
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			in := valid()
			test.change(&in)
			if _, _, err := validateComment(in); err == nil {
				t.Fatal("unsafe or out-of-bounds document accepted")
			}
		})
	}
}

func TestRichCommentDerivesTextAndUniqueRecipients(t *testing.T) {
	in := CommentInput{SchemaVersion: 1, Content: &models.CommentContent{Blocks: []models.CommentBlock{
		{Type: "paragraph", Children: []models.CommentInline{{Type: "text", Text: "Hello "}, {Type: "mention", UserID: "person"}}},
		{Type: "bullet_list", Children: []models.CommentInline{{Type: "text", Text: "First finding"}}},
		{Type: "ordered_list", Children: []models.CommentInline{{Type: "text", Text: "Second finding "}, {Type: "mention", UserID: "person"}}},
	}}}
	content, ids, err := validateComment(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "person" {
		t.Fatalf("recipients not deduplicated: %v", ids)
	}
	text := commentText(content, map[string]models.User{"person": {Name: "Actual Name"}})
	if text != "Hello @Actual Name\nFirst finding\nSecond finding @Actual Name" {
		t.Fatalf("text lost block separation/identity: %q", text)
	}
	legacy := "Existing plain text\nwith another line"
	content, _, err = validateComment(CommentInput{Comment: &legacy})
	if err != nil {
		t.Fatal(err)
	}
	if commentText(content, nil) != legacy {
		t.Fatal("legacy text was not preserved")
	}
}
