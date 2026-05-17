package web

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
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
	button { font-family: inherit; background: none; border: none; cursor: pointer; padding: 0; color: gray; }
	.capture-nav { display: flex; align-items: center; gap: 1em; padding: 0.5em 1em; border-bottom: 1px solid #ccc; }
	.capture-nav a { text-decoration: none; color: inherit; white-space: nowrap; }
	.capture-nav a:hover { text-decoration: underline; }
	.note-list { padding: 1em; display: flex; flex-direction: column; gap: 0.5em; }
	.note-item { padding: 0.25em 0; border-bottom: 1px solid #eee; }
`)

func captureNav() g.Node {
	return h.Nav(h.Class("capture-nav"),
		h.A(h.Href("/capture/tasks"), g.Text("tasks")),
		h.A(h.Href("/capture/reference"), g.Text("reference")),
		h.Form(
			g.Attr("data-on:submit", "@post('/capture')"),
			h.Style("flex:1; margin:0"),
			h.Input(
				g.Attr("data-bind", "body"),
				h.Style("width:100%"),
				h.Placeholder("capture..."),
			),
		),
	)
}

func capturePage(body g.Node) g.Node {
	return h.HTML(
		h.Head(
			h.Script(h.Type("module"), h.Src("https://cdn.jsdelivr.net/gh/starfederation/datastar@1.0.0-RC.7/bundles/datastar.js")),
			h.StyleEl(captureStyles),
		),
		h.Body(
			g.Attr("data-signals", `{"body":""}`),
			body,
		),
	)
}

func (s *webservice) captureIndex(w http.ResponseWriter, r *http.Request) {
	capturePage(captureNav()).Render(w)
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
	owner := getUserInfo(r).Id

	scheduled, err := s.app.Notes.FindAllByCategoryAndSubcategory(owner, "task", "scheduled")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.SortStableFunc(scheduled, func(a, b note.Note) int {
		if a.Due != nil && b.Due != nil {
			return int(*a.Due - *b.Due)
		}
		return 0
	})

	someday, err := s.app.Notes.FindAllByCategoryAndSubcategory(owner, "task", "someday")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(someday)

	notnow, err := s.app.Notes.FindAllByCategoryAndSubcategory(owner, "task", "notnow")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(notnow)

	capturePage(g.Group{
		captureNav(),
		captureNotnowSection(notnow),
		g.Group(g.Map(partitionScheduled(scheduled), func(b scheduledBucket) g.Node {
			return captureTaskSection(b.name, b.notes)
		})),
		captureSomedaySection(someday),
	}).Render(w)
}

type scheduledBucket struct {
	name  string
	notes []note.Note
}

func partitionScheduled(notes []note.Note) []scheduledBucket {
	midnight := notesmeta.Midnight(time.Now())

	buckets := []scheduledBucket{{name: "overdue"}}
	for _, tf := range notesmeta.TimeframeList {
		buckets = append(buckets, scheduledBucket{name: tf.DisplayName})
	}
	buckets = append(buckets, scheduledBucket{name: "later"})

	for _, n := range notes {
		if n.Due == nil {
			continue
		}
		due := *n.Due
		if due < midnight.Unix() {
			buckets[0].notes = append(buckets[0].notes, n)
			continue
		}
		placed := false
		for i, tf := range notesmeta.TimeframeList {
			if due <= midnight.AddDate(0, 0, tf.Days()).Unix() {
				buckets[i+1].notes = append(buckets[i+1].notes, n)
				placed = true
				break
			}
		}
		if !placed {
			buckets[len(buckets)-1].notes = append(buckets[len(buckets)-1].notes, n)
		}
	}

	return buckets
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

func (s *webservice) postCaptureTransition(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = s.app.Commander.Send(commands.TransitionNoteSubcategory{
		NoteID:          noteID,
		TransitionEvent: chi.URLParam(r, "event"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/capture/tasks", http.StatusSeeOther)
}

func scheduleButtons(n note.Note) g.Node {
	transitionBtn := func(event, label string) g.Node {
		return h.Form(
			h.Method("POST"),
			h.Action(fmt.Sprintf("/capture/trans/%s/%s", n.ID, event)),
			h.Style("display:inline"),
			h.Button(h.Type("submit"), h.Style("padding:0 0.25em"), g.Text(label)),
		)
	}
	return h.Span(h.Style("margin-left:0.5em"),
		g.Map(notesmeta.TimeframeList, func(tf notesmeta.Timeframe) g.Node {
			return transitionBtn(tf.EventName, tf.DisplayName)
		}),
		transitionBtn("someday", "Someday"),
	)
}

func captureNotnowSection(noteList []note.Note) g.Node {
	if len(noteList) == 0 {
		return nil
	}
	return h.Div(
		h.Div(h.Style("padding: 0 1em; margin: 0.5em 0 0.25em; color: gray"), g.Text("not scheduled")),
		h.Div(h.Class("note-list"),
			g.Map(noteList, func(n note.Note) g.Node {
				return h.Div(h.Class("note-item"),
					h.Span(g.Text(n.Text)),
					scheduleButtons(n),
				)
			}),
		),
	)
}

func dueRemaining(dueUnix int64) string {
	due := time.Unix(dueUnix, 0)
	now := time.Now()
	dy, dm, dd := due.Date()
	ny, nm, nd := now.Date()
	if dy == ny && dm == nm && dd == nd {
		h := int(time.Until(due).Hours())
		return fmt.Sprintf("%dh", h)
	}
	days := int(time.Until(due).Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

func captureNoteList(noteList []note.Note) g.Node {
	return h.Div(h.Class("note-list"),
		g.Map(noteList, func(n note.Note) g.Node {
			return h.Div(h.Class("note-item"), g.Text(n.Text))
		}),
	)
}

func rescheduleBtn(n note.Note, label string) g.Node {
	return h.Form(
		h.Method("POST"),
		h.Action(fmt.Sprintf("/capture/trans/%s/reschedule", n.ID)),
		h.Style("display:inline; margin-left:0.5em"),
		h.Button(h.Type("submit"), h.Style("color:gray; padding:0"), g.Text("· "+label)),
	)
}

func captureSomedaySection(noteList []note.Note) g.Node {
	if len(noteList) == 0 {
		return nil
	}
	return h.Div(
		h.Div(h.Style("padding: 0 1em; margin: 0.5em 0 0.25em; color: gray"), g.Text("someday")),
		h.Div(h.Class("note-list"),
			g.Map(noteList, func(n note.Note) g.Node {
				return h.Div(h.Class("note-item"),
					h.Span(g.Text(n.Text)),
					rescheduleBtn(n, "someday"),
				)
			}),
		),
	)
}

func captureTaskSection(heading string, noteList []note.Note) g.Node {
	if len(noteList) == 0 {
		return nil
	}
	return h.Div(
		h.Div(h.Style("padding: 0 1em; margin: 0.5em 0 0.25em; color: gray"), g.Text(heading)),
		h.Div(h.Class("note-list"),
			g.Map(noteList, func(n note.Note) g.Node {
				return h.Div(h.Class("note-item"),
					h.Span(g.Text(n.Text)),
					g.Iff(n.Due != nil, func() g.Node {
						return rescheduleBtn(n, dueRemaining(*n.Due))
					}),
				)
			}),
		),
	)
}
