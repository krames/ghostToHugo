package ghostToHugo

import (
	"bytes"
	"encoding/json"
	"path"
	"strings"
	"time"

	"github.com/gohugoio/hugo/hugolib"
	"github.com/spf13/viper"
)

type mobiledocCard struct {
	Name     string `json:"cardName"`
	Markdown string `json:"markdown"`
}

type post struct {
	ID              json.RawMessage `json:"id"`
	Title           string          `json:"title"`
	Slug            string          `json:"slug"`
	Content         string          `json:"markdown"`
	Plain           string          `json:"plaintext"`
	MobileDoc       string          `json:"mobiledoc",omitempty`
	Image           string          `json:"image"`
	Page            json.RawMessage `json:"page"`
	Status          string          `json:"status"`
	MetaDescription string          `json:"meta_description"`
	AuthorID        json.RawMessage `json:"author_id"`
	PublishedAt     time.Time       `json:"published_at"`
	CreatedAt       time.Time       `json:"created_at"`

	IsDraft bool
	IsPage  bool
	Author  string
	Tags    []string
}

func (p *post) populate(gi *ghostInfo, gth *GhostToHugo) {
	p.IsDraft = p.Status == "draft"
	p.IsPage = parseBool(p.Page)

	for _, user := range gi.users {
		if bytes.Equal(user.ID, p.AuthorID) {
			p.Author = user.Name
			break
		}
	}

	for _, pt := range gi.posttags {
		if bytes.Equal(pt.PostID, p.ID) {
			for _, t := range gi.tags {
				if bytes.Equal(t.ID, pt.TagID) {
					p.Tags = append(p.Tags, strings.TrimPrefix(t.Name, "#"))
					break
				}
			}
		}
	}
}

func (p post) mobiledocMarkdown() string {
	if p.MobileDoc == "" {
		return ""
	}

	decoder := json.NewDecoder(bytes.NewReader([]byte(p.MobileDoc)))
	err := seekTo(decoder, "cards")
	if err != nil {
		return ""
	}
	_, err = decoder.Token() // Stip token
	if err != nil {
		return ""
	}

	for decoder.More() {

		_, err = decoder.Token() // Stip token
		if err != nil {
			return ""
		}

		_, err = decoder.Token() // Stip token
		if err != nil {
			return ""
		}

		var card mobiledocCard
		err = decoder.Decode(&card)
		if err != nil {
			return ""
		}

		if card.Name == "card-markdown" {
			return card.Markdown
		}
	}

	return ""
}

func (p post) path(site *hugolib.Site) string {
	if p.IsPage {
		return site.PathSpec.AbsPathify(
			path.Join(viper.GetString("contentDir"), p.Slug+".md"))
	}

	return site.PathSpec.AbsPathify(
		path.Join(viper.GetString("contentDir"), "post", p.Slug+".md"))
}

func stripContentFolder(originalString string) string {
	return strings.Replace(originalString, "/content/", "/", -1)
}

func (p post) metadata() map[string]interface{} {
	metadata := make(map[string]interface{})

	switch p.IsDraft {
	case true:
		metadata["date"] = p.CreatedAt
	case false:
		metadata["date"] = p.PublishedAt
	}
	metadata["title"] = p.Title
	metadata["draft"] = p.IsDraft
	metadata["slug"] = p.Slug
	metadata["description"] = p.MetaDescription
	if p.Image != "" {
		metadata["image"] = stripContentFolder(p.Image)
	}
	if len(p.Tags) > 0 {
		metadata["tags"] = p.Tags
		metadata["categories"] = p.Tags
	}
	if p.Author != "" {
		metadata["author"] = p.Author
	}

	return metadata
}
