package services

import (
	"strings"
	"testing"
	"zyra-api/internal/models"
)

func gifDocument(nodes ...models.CommentInline) CommentInput {
	return CommentInput{SchemaVersion: 1, Content: &models.CommentContent{Blocks: []models.CommentBlock{{Type: "paragraph", Children: nodes}}}}
}

func TestGIFCommentValidationAndProjection(t *testing.T) {
	for _, src := range []string{"https://images.example.test/fix.gif", "https://images.example.test/animation?id=123"} {
		in := gifDocument(models.CommentInline{Type: "gif", Src: src, Alt: "Fixed the backup"})
		content, recipients, err := validateComment(in)
		if err != nil {
			t.Fatalf("valid GIF-only comment rejected: %v", err)
		}
		if len(recipients) != 0 || commentText(content, nil) != "[GIF: Fixed the backup]" {
			t.Fatalf("incorrect projection or mention extraction: %q %v", commentText(content, nil), recipients)
		}
	}
	content, _, err := validateComment(gifDocument(models.CommentInline{Type: "gif", Src: "https://images.example.test/a.gif"}))
	if err != nil || commentText(content, nil) != "[GIF]" {
		t.Fatalf("GIF without alt should have readable fallback: %v", err)
	}
	content, recipients, err := validateComment(gifDocument(
		models.CommentInline{Type: "gif", Src: "https://images.example.test/@forged.gif", Alt: "@forged"},
		models.CommentInline{Type: "mention", UserID: "actual"},
		models.CommentInline{Type: "mention", UserID: "actual"},
	))
	if err != nil || len(recipients) != 1 || recipients[0] != "actual" || !strings.Contains(commentText(content, map[string]models.User{"actual": {Name: "Real Person"}}), "@Real Person") {
		t.Fatalf("GIF fields must not create mention identities: %v %v", recipients, err)
	}
}

func TestGIFCommentRejectsUnsafeAndForeignFields(t *testing.T) {
	valid := func() models.CommentInline {
		return models.CommentInline{Type: "gif", Src: "https://images.example.test/a.gif"}
	}
	for _, src := range []string{"", "http://images.example.test/a.gif", "//images.example.test/a.gif", "javascript:alert(1)", "data:image/gif;base64,AAAA", "https://user:password@images.example.test/a.gif", "https://", "https://images.example.test/" + strings.Repeat("x", 2048)} {
		t.Run(src[:min(len(src), 50)], func(t *testing.T) {
			node := valid()
			node.Src = src
			if _, _, err := validateComment(gifDocument(node)); err == nil {
				t.Fatal("invalid GIF source accepted")
			}
		})
	}
	for _, change := range []func(*models.CommentInline){
		func(n *models.CommentInline) { n.Text = "forged" },
		func(n *models.CommentInline) { n.UserID = "forged" },
		func(n *models.CommentInline) { n.Href = "https://example.test" },
		func(n *models.CommentInline) { n.Bold = true },
		func(n *models.CommentInline) { n.Color = "#ffffff" },
		func(n *models.CommentInline) { n.Size = 16 },
		func(n *models.CommentInline) { n.Alt = strings.Repeat("x", 201) },
	} {
		node := valid()
		change(&node)
		if _, _, err := validateComment(gifDocument(node)); err == nil {
			t.Fatal("GIF accepted foreign or excessive fields")
		}
	}
	for _, node := range []models.CommentInline{{Type: "text", Text: "hello", Src: "https://example.test/a.gif"}, {Type: "text", Text: "hello", Alt: "foreign"}, {Type: "mention", UserID: "person", Src: "https://example.test/a.gif"}, {Type: "mention", UserID: "person", Alt: "foreign"}, {Type: "image", Src: "https://example.test/a.png"}} {
		if _, _, err := validateComment(gifDocument(node)); err == nil {
			t.Fatalf("unsupported media fields/type accepted: %#v", node)
		}
	}
	nodes := make([]models.CommentInline, 10)
	for i := range nodes {
		nodes[i] = valid()
	}
	if _, _, err := validateComment(gifDocument(nodes...)); err != nil {
		t.Fatal(err)
	}
	nodes = append(nodes, valid())
	if _, _, err := validateComment(gifDocument(nodes...)); err == nil {
		t.Fatal("more than ten GIFs accepted")
	}
}
