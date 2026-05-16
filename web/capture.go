package web

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/catalog/notesmeta"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func (s *webservice) captureIndex(w http.ResponseWriter, r *http.Request) {
	h.HTML(
		h.Head(
			h.Script(h.Type("module"), h.Src("https://cdn.jsdelivr.net/gh/starfederation/datastar@1.0.0-RC.7/bundles/datastar.js")),
			h.StyleEl(g.Raw(styles)),
			h.StyleEl(g.Raw(`
				body { margin: 0; height: 100vh; display: flex; align-items: center; justify-content: center; }
				#capture-input { font-size: 1.5em; padding: 0.5em; width: 500px; max-width: 90vw; }
			`)),
		),
		h.Body(
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
