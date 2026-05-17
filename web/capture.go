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
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

var captureStyles = g.Raw(`
	body { margin: 0; font-family: monospace; }
	button { font-family: inherit; background: none; border: none; cursor: pointer; padding: 0; color: gray; vertical-align: baseline; }
	.capture-nav { display: flex; align-items: center; gap: 1em; padding: 0.5em 1em; border-bottom: 1px solid #ccc; position: sticky; top: 0; background: white; z-index: 1; }
	.capture-nav a { text-decoration: none; color: inherit; white-space: nowrap; }
	.capture-nav a:hover { text-decoration: underline; }
	.note-list { padding: 1em; display: flex; flex-direction: column; gap: 0.5em; }
	.note-item { padding: 0.25em 0; border-bottom: 1px solid #eee; }
	.invisible { visibility: hidden; }
	details > summary { padding: 0 1em; margin: 0.5em 0 0.25em; font-weight: bold; cursor: pointer; font-size: inherit; font-family: inherit; }
	details > summary::-webkit-details-marker, details > summary::marker { color: #ccc; }
`)

func captureNavWithRequest(r *http.Request, postAction string) g.Node {
	return captureNav(postAction, getUserInfo(r).Picture)
}

func captureNav(postAction, pictureURL string) g.Node {
	return h.Nav(h.Class("capture-nav"),
		h.A(h.Href("/capture/tasks"), g.Text("tasks")),
		h.A(h.Href("/capture/reference"), g.Text("reference")),
		h.Form(
			h.Method("POST"),
			h.Action(postAction),
			h.Style("flex:1; margin:0"),
			h.Input(
				h.Name("body"),
				h.Style("width:100%"),
				h.Placeholder(func() string {
					if postAction == "/capture/tasks" {
						return "capture task..."
					}
					return "capture thing to remember..."
				}()),
				h.AutoComplete("off"),
			),
		),
		h.Img(h.Src(pictureURL), h.Style("width:1.5em; height:1.5em; border-radius:50%")),
	)
}

func capturePage(body g.Node) g.Node {
	return h.HTML(
		h.Head(
			h.Script(h.Type("module"), h.Src("https://cdn.jsdelivr.net/gh/starfederation/datastar@1.0.0-RC.7/bundles/datastar.js")),
			h.StyleEl(captureStyles),
		),
		h.Body(
			g.Attr("data-signals", `{"activeNote":""}`),
			body,
		),
	)
}

func (s *webservice) captureIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/capture/tasks", http.StatusSeeOther)
}

func (s *webservice) postCaptureTask(w http.ResponseWriter, r *http.Request) {
	userInfo := getUserInfo(r)
	if body := r.FormValue("body"); body != "" {
		err := s.app.Commander.Send(commands.CreateNote{
			Owner:       userInfo.Id,
			NoteID:      uuid.New(),
			Text:        body,
			Category:    notesmeta.Task.Slug,
			Subcategory: notesmeta.Task.Inbox().Slug,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/capture/tasks", http.StatusSeeOther)
}

func (s *webservice) postCaptureReference(w http.ResponseWriter, r *http.Request) {
	userInfo := getUserInfo(r)
	if body := r.FormValue("body"); body != "" {
		err := s.app.Commander.Send(commands.CreateNote{
			Owner:       userInfo.Id,
			NoteID:      uuid.New(),
			Text:        body,
			Category:    notesmeta.Note.Slug,
			Subcategory: notesmeta.Note.Inbox().Slug,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/capture/reference", http.StatusSeeOther)
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

	done, err := s.app.Notes.FindAllByCategoryAndSubcategory(owner, "task", "done")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(done)

	capturePage(g.Group{
		captureNavWithRequest(r, "/capture/tasks"),
		captureNotnowSection(notnow),
		g.Group(g.Map(partitionScheduled(scheduled), func(b scheduledBucket) g.Node {
			return captureTaskSection(b.name, b.notes)
		})),
		captureSomedaySection(someday),
		captureDoneSection(done),
	}).Render(w)
}

type scheduledBucket struct {
	name  string
	notes []note.Note
}

func partitionScheduled(notes []note.Note) []scheduledBucket {
	loc, _ := time.LoadLocation("America/Vancouver")
	now := time.Now().In(loc)
	midnight := notesmeta.Midnight(now)

	buckets := []scheduledBucket{{name: "Overdue"}}
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
			if due <= midnight.AddDate(0, 0, tf.Days(now)).Unix() {
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
		captureNavWithRequest(r, "/capture/reference"),
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
		transitionBtn("done", "Never"),
		transitionBtn("done", "Done"),
	)
}

func captureNotnowSection(noteList []note.Note) g.Node {
	if len(noteList) == 0 {
		return nil
	}
	return h.Div(
		h.Div(h.Style("padding: 0 1em; margin: 0.5em 0 0.25em; font-weight: bold"), g.Text("Unscheduled")),
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

func captureNoteList(noteList []note.Note) g.Node {
	return h.Div(h.Class("note-list"),
		g.Map(noteList, func(n note.Note) g.Node {
			return h.Div(h.Class("note-item"), g.Text(n.Text))
		}),
	)
}

func noteActions(n note.Note) g.Node {
	includeDone := n.Subcategory != "done"
	includeReschedule := n.Subcategory != "notnow"
	actionBtn := func(event, label string) g.Node {
		return h.Form(
			h.Method("POST"),
			h.Action(fmt.Sprintf("/capture/trans/%s/%s", n.ID, event)),
			h.Style("display:inline"),
			h.Button(h.Type("submit"), h.Style("color:gray; padding:0"), g.Text(label)),
		)
	}
	return h.Span(
		g.Attr("data-class", fmt.Sprintf(`{"invisible": $activeNote !== '%s'}`, n.ID)),
		h.Class("invisible"),
		h.Style("margin-left:0.5em"),
		g.If(includeDone, actionBtn("done", "done")),
		g.If(includeDone && includeReschedule, g.Text(" · ")),
		g.If(includeReschedule, actionBtn("reschedule", "reschedule")),
	)
}

func captureDoneSection(noteList []note.Note) g.Node {
	if len(noteList) == 0 {
		return nil
	}
	return h.Details(
		h.Summary(g.Text("Done")),
		h.Div(h.Class("note-list"),
			g.Map(noteList, func(n note.Note) g.Node {
				return h.Div(h.Class("note-item"),
					h.Span(g.Attr("data-on:click", fmt.Sprintf("$activeNote = $activeNote === '%s' ? '' : '%s'", n.ID, n.ID)), h.Style("cursor:pointer"), g.Text(n.Text)),
					noteActions(n),
				)
			}),
		),
	)
}

func captureSomedaySection(noteList []note.Note) g.Node {
	if len(noteList) == 0 {
		return nil
	}
	return h.Details(
		h.Summary(g.Text("Someday")),
		h.Div(h.Class("note-list"),
			g.Map(noteList, func(n note.Note) g.Node {
				return h.Div(h.Class("note-item"),
					h.Span(g.Attr("data-on:click", fmt.Sprintf("$activeNote = $activeNote === '%s' ? '' : '%s'", n.ID, n.ID)), h.Style("cursor:pointer"), g.Text(n.Text)),
					noteActions(n),
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
		h.Div(h.Style("padding: 0 1em; margin: 0.5em 0 0.25em; font-weight: bold"), g.Text(heading)),
		h.Div(h.Class("note-list"),
			g.Map(noteList, func(n note.Note) g.Node {
				return h.Div(h.Class("note-item"),
					h.Span(g.Attr("data-on:click", fmt.Sprintf("$activeNote = $activeNote === '%s' ? '' : '%s'", n.ID, n.ID)), h.Style("cursor:pointer"), g.Text(n.Text)),
					noteActions(n),
				)
			}),
		),
	)
}
