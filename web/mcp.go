package web

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcy/whatever/catalog/notesmeta"
	"github.com/rcy/whatever/projections/note"
)

// mcpNote is the shape of a note returned to MCP clients.
type mcpNote struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Category    string   `json:"category"`
	Subcategory string   `json:"subcategory"`
	State       string   `json:"state"`
	Starred     bool     `json:"starred"`
	Tags        []string `json:"tags,omitempty"`
	CreatedAt   string   `json:"created_at"`
	Due         string   `json:"due,omitempty"`
}

type listNotesInput struct {
	Query       string `json:"query,omitempty" jsonschema:"case-insensitive substring to match against note text"`
	Category    string `json:"category,omitempty" jsonschema:"filter by category slug (see notes_overview)"`
	Subcategory string `json:"subcategory,omitempty" jsonschema:"filter by subcategory slug"`
	Tag         string `json:"tag,omitempty" jsonschema:"only notes with this tag"`
	Person      string `json:"person,omitempty" jsonschema:"only notes mentioning this @handle (without the @)"`
	Starred     bool   `json:"starred,omitempty" jsonschema:"only starred notes"`
	Limit       int    `json:"limit,omitempty" jsonschema:"max notes to return, newest first (default 50)"`
}

type listNotesOutput struct {
	Total int       `json:"total" jsonschema:"number of notes matching the filters before the limit was applied"`
	Notes []mcpNote `json:"notes"`
}

type getNoteInput struct {
	ID string `json:"id" jsonschema:"the note id"`
}

type overviewInput struct{}

type mcpCategory struct {
	Slug          string           `json:"slug"`
	Name          string           `json:"name"`
	Subcategories []mcpSubcategory `json:"subcategories"`
}

type mcpSubcategory struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type overviewOutput struct {
	Categories []mcpCategory `json:"categories"`
	Tags       []string      `json:"tags"`
	People     []string      `json:"people"`
}

func (s *webservice) mcpHandler() http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "whatever", Version: "1.0.0"}, &mcp.ServerOptions{
		Instructions: "Read-only access to the owner's notes. Call notes_overview first to learn the category, subcategory, tag and people slugs, then list_notes to filter or search.",
	})

	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "notes_overview",
		Description: "List note categories and subcategories (with counts), all tags, and all mentioned people.",
		Annotations: readOnly,
	}, s.mcpOverview)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_notes",
		Description: "List or search notes, newest first. All filters are optional and combine with AND.",
		Annotations: readOnly,
	}, s.mcpListNotes)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_note",
		Description: "Get a single note by id.",
		Annotations: readOnly,
	}, s.mcpGetNote)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true})
	return s.oauth.requireToken(handler)
}

func (s *webservice) mcpOverview(ctx context.Context, req *mcp.CallToolRequest, _ overviewInput) (*mcp.CallToolResult, overviewOutput, error) {
	var out overviewOutput
	for _, cat := range notesmeta.Categories {
		counts, err := s.app.Notes.SubcategoryCounts(s.ownerID, cat.Slug)
		if err != nil {
			return nil, out, err
		}
		c := mcpCategory{Slug: cat.Slug, Name: cat.DisplayName}
		for _, sub := range cat.Subcategories {
			n := 0
			for _, sc := range counts {
				if sc.Subcategory == sub.Slug {
					n = sc.Count
				}
			}
			c.Subcategories = append(c.Subcategories, mcpSubcategory{Slug: sub.Slug, Name: sub.DisplayName, Count: n})
		}
		out.Categories = append(out.Categories, c)
	}

	var err error
	if out.Tags, err = s.app.Notes.FindAllTagsForOwner(s.ownerID); err != nil {
		return nil, out, err
	}
	if out.People, err = s.app.Notes.FindAllPeople(s.ownerID); err != nil {
		return nil, out, err
	}
	return nil, out, nil
}

func (s *webservice) mcpListNotes(ctx context.Context, req *mcp.CallToolRequest, in listNotesInput) (*mcp.CallToolResult, listNotesOutput, error) {
	var (
		notes []note.Note
		err   error
	)
	if in.Person != "" {
		notes, err = s.app.Notes.FindAllByPerson(s.ownerID, strings.TrimPrefix(in.Person, "@"))
	} else {
		notes, err = s.app.Notes.FindAll(s.ownerID)
	}
	if err != nil {
		return nil, listNotesOutput{}, err
	}

	query := strings.ToLower(in.Query)
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}

	out := listNotesOutput{Notes: []mcpNote{}}
	// projection returns oldest first; walk backwards for newest first
	for _, n := range slices.Backward(notes) {
		if in.Category != "" && n.Category != in.Category {
			continue
		}
		if in.Subcategory != "" && n.Subcategory != in.Subcategory {
			continue
		}
		if in.Starred && !n.Starred {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(n.Text), query) {
			continue
		}
		mn, err := s.toMCPNote(n)
		if err != nil {
			return nil, listNotesOutput{}, err
		}
		if in.Tag != "" && !slices.Contains(mn.Tags, in.Tag) {
			continue
		}
		out.Total++
		if len(out.Notes) < limit {
			out.Notes = append(out.Notes, mn)
		}
	}
	return nil, out, nil
}

func (s *webservice) mcpGetNote(ctx context.Context, req *mcp.CallToolRequest, in getNoteInput) (*mcp.CallToolResult, mcpNote, error) {
	n, err := s.app.Notes.FindOne(in.ID)
	if err != nil || n.Owner != s.ownerID {
		return nil, mcpNote{}, fmt.Errorf("note %q not found", in.ID)
	}
	mn, err := s.toMCPNote(n)
	return nil, mn, err
}

func (s *webservice) toMCPNote(n note.Note) (mcpNote, error) {
	tags, err := s.app.Notes.FindTagsForNote(n.ID)
	if err != nil {
		return mcpNote{}, err
	}
	mn := mcpNote{
		ID:          n.ID.String(),
		Text:        n.Text,
		Category:    n.Category,
		Subcategory: n.Subcategory,
		State:       n.State,
		Starred:     n.Starred,
		Tags:        tags,
		CreatedAt:   time.Unix(n.Ts, 0).UTC().Format(time.RFC3339),
	}
	if n.Due != nil {
		mn.Due = time.Unix(*n.Due, 0).UTC().Format(time.RFC3339)
	}
	return mn, nil
}
