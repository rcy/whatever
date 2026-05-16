package web

import (
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/rcy/whatever/catalog/notesmeta"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/projections/note"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

var captureStyles = g.Raw(`
	body { margin: 0; font-family: monospace; }
	.capture-nav { display: flex; gap: 1em; padding: 1em; border-bottom: 1px solid #ccc; }
	.capture-nav a { text-decoration: none; color: inherit; }
	.capture-nav a:hover { text-decoration: underline; }
	.capture-input-wrap { display: flex; justify-content: center; padding: 2em; }
	#capture-input { font-size: 1.5em; padding: 0.5em; width: 500px; max-width: 90vw; }
	.note-list { padding: 1em; display: flex; flex-direction: column; gap: 0.5em; }
	.note-item { padding: 0.25em 0; border-bottom: 1px solid #eee; }
`)

func captureNav() g.Node {
	return h.Nav(h.Class("capture-nav"),
		h.A(h.Href("/capture"), g.Text("capture")),
		h.A(h.Href("/capture/tasks"), g.Text("tasks")),
		h.A(h.Href("/capture/reference"), g.Text("reference")),
	)
}

func capturePage(body g.Node) g.Node {
	return h.HTML(
		h.Head(
			h.Script(h.Type("module"), h.Src("https://cdn.jsdelivr.net/gh/starfederation/datastar@1.0.0-RC.7/bundles/datastar.js")),
			h.StyleEl(captureStyles),
		),
		h.Body(body),
	)
}

func (s *webservice) captureIndex(w http.ResponseWriter, r *http.Request) {
	capturePage(
		g.Group{
			captureNav(),
			h.Div(h.Class("capture-input-wrap"),
				g.Attr("data-signals", `{"body":""}`),
				h.Form(
					g.Attr("data-on:submit", "@post('/capture')"),
					h.Style("margin:0"),
					h.Input(
						h.ID("capture-input"),
						g.Attr("data-bind", "body"),
						h.Placeholder("What's on your mind?"),
						h.AutoFocus(),
					),
				),
			),
		},
	).Render(w)
}

func (s *webservice) postCapture(w http.ResponseWriter, r *http.Request) {
	userInfo := getUserInfo(r)

	var sig signals
	if err := datastar.ReadSignals(r, &sig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if sig.Body != "" {
		err := s.app.Commander.Send(commands.CreateNote{
			Owner:       userInfo.Id,
			NoteID:      uuid.New(),
			Text:        sig.Body,
			Category:    notesmeta.Inbox.Slug,
			Subcategory: notesmeta.Inbox.Inbox().Slug,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	sig.Body = ""
	sse := datastar.NewSSE(w, r)
	sse.MarshalAndPatchSignals(sig)
}

func (s *webservice) captureTasksIndex(w http.ResponseWriter, r *http.Request) {
	userInfo := getUserInfo(r)
	noteList, err := s.app.Notes.FindAllByCategory(userInfo.Id, "task")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(noteList)
	capturePage(g.Group{
		captureNav(),
		captureNoteList(noteList),
	}).Render(w)
}

func (s *webservice) captureReferenceIndex(w http.ResponseWriter, r *http.Request) {
	userInfo := getUserInfo(r)
	noteList, err := s.app.Notes.FindAllByCategory(userInfo.Id, "reference")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(noteList)
	capturePage(g.Group{
		captureNav(),
		captureNoteList(noteList),
	}).Render(w)
}

func captureNoteList(noteList []note.Note) g.Node {
	return h.Div(h.Class("note-list"),
		g.Map(noteList, func(n note.Note) g.Node {
			return h.Div(h.Class("note-item"), g.Text(n.Text))
		}),
	)
}
