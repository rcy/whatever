package web

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
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
	body { margin: 0; font-family: monospace; font-size: 16px; }
	button { font-family: inherit; font-size: inherit; background: none; border: none; cursor: pointer; padding: 0.25em 0.5em; color: gray; vertical-align: baseline; }
	.capture-nav { display: flex; align-items: center; gap: 1em; padding: 0.5em 1em; border-bottom: 1px solid #ccc; position: sticky; top: 0; background: #f5f5f5; z-index: 1; }
	.capture-nav a { text-decoration: none; color: inherit; white-space: nowrap; }
	.capture-nav a:hover { text-decoration: underline; }
	.capture-nav input { font-size: 16px; }
	.note-list { padding: 1em; display: flex; flex-direction: column; gap: 0.5em; }
	.note-item { padding: 0.5em 0; border-bottom: 1px solid #eee; }
	.invisible { visibility: hidden; }
	details > summary { padding: 0 1em; margin: 0.5em 0 0.25em; font-weight: bold; cursor: pointer; font-size: inherit; font-family: inherit; }
	details > summary::-webkit-details-marker, details > summary::marker { color: #ccc; }
`)

func captureNavWithRequest(r *http.Request, postAction string) g.Node {
	return captureNav(postAction, getUserInfo(r).Picture, r.URL.Path)
}

func captureNav(postAction, pictureURL, activePath string) g.Node {
	navItem := func(href, label string) g.Node {
		if activePath == href {
			return h.Span(h.Style("font-weight:bold"), g.Text(label))
		}
		return h.A(h.Href(href), g.Text(label))
	}
	return h.Nav(h.Class("capture-nav"),
		h.Span(h.Style("font-weight:bold"), g.Text("NOTNOW")),
		navItem("/capture/tasks", "tasks"),
		navItem("/capture/reference", "notes"),
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
					return "capture note..."
				}()),
				h.AutoComplete("off"),
			),
		),
		h.A(h.Href("/settings"), h.Style("display:flex; align-items:center"),
			h.Img(h.Src(pictureURL), h.Style("width:1.5em; height:1.5em; border-radius:50%")),
		),
	)
}

func capturePage(body g.Node) g.Node {
	return h.HTML(
		h.Head(
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
			h.Script(h.Type("module"), h.Src("https://cdn.jsdelivr.net/gh/starfederation/datastar@1.0.0-RC.7/bundles/datastar.js")),
			h.StyleEl(captureStyles),
		),
		h.Body(
			g.Attr("data-signals", `{"activeNote":"","editNote":""}`),
			body,
		),
	)
}

func (s *webservice) captureIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/capture/reference", http.StatusSeeOther)
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
			if b.overdue {
				return captureOverdueSection(b.notes)
			}
			return captureTaskSection(b.name, b.notes, b.today)
		})),
		captureSomedaySection(someday),
		captureDoneSection(done),
	}).Render(w)
}

type scheduledBucket struct {
	name    string
	notes   []note.Note
	overdue bool
	today   bool
}

func partitionScheduled(notes []note.Note) []scheduledBucket {
	loc, _ := time.LoadLocation("America/Vancouver")
	now := time.Now().In(loc)
	midnight := notesmeta.Midnight(now)

	buckets := []scheduledBucket{{name: "Overdue", overdue: true}}
	for i, tf := range notesmeta.TimeframeList {
		buckets = append(buckets, scheduledBucket{name: tf.DisplayName, today: i == 0})
	}
	buckets = append(buckets, scheduledBucket{name: "later"})

	for _, n := range notes {
		if n.Due == nil {
			continue
		}
		due := *n.Due
		if due < midnight.AddDate(0, 0, 1).Unix() {
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
	selectedTag := r.URL.Query().Get("tag")
	if r.URL.Query().Get("untagged") == "1" {
		selectedTag = filterUntagged
	}

	var noteList []note.Note
	var err error
	switch {
	case selectedTag == filterUntagged:
		noteList, err = s.app.Notes.FindUntaggedByCategory(userInfo.Id, "reference")
	case selectedTag != "":
		noteList, err = s.app.Notes.FindAllByCategoryAndTag(userInfo.Id, "reference", selectedTag)
	default:
		noteList, err = s.app.Notes.FindAllByCategory(userInfo.Id, "reference")
		slices.Reverse(noteList)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	noteTags, err := s.app.Notes.FindTagsByOwnerAndCategory(userInfo.Id, "reference")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tagMap := map[uuid.UUID][]string{}
	for _, nt := range noteTags {
		id, _ := uuid.Parse(nt.NoteID)
		tagMap[id] = append(tagMap[id], nt.Tag)
	}

	allTags, err := s.app.Notes.FindAllTagsForOwner(userInfo.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	capturePage(g.Group{
		captureNavWithRequest(r, "/capture/reference"),
		tagHeader(allTags, selectedTag),
		captureNoteList(noteList, tagMap),
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

func (s *webservice) postCaptureStar(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	n, err := s.app.Notes.FindOne(noteID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var cmd interface{ AggregateID() uuid.UUID }
	if n.Starred {
		cmd = commands.UnstarNote{NoteID: noteID}
	} else {
		cmd = commands.StarNote{NoteID: noteID}
	}
	if err := s.app.Commander.Send(cmd); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/capture/tasks", http.StatusSeeOther)
}

func starButton(n note.Note) g.Node {
	star := "☆"
	color := "black"
	if n.Starred {
		star = "★"
		color = "red"
	}
	return h.Form(
		h.Method("POST"),
		h.Action(fmt.Sprintf("/capture/notes/%s/star", n.ID)),
		h.Style("display:inline"),
		h.Button(h.Type("submit"), h.Style(fmt.Sprintf("padding:0 0.25em; color:%s; font-size:1.2em; line-height:1; font-family:sans-serif", color)), g.Text(star)),
	)
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

func inlineEditForm(n note.Note) g.Node {
	return h.Form(
		h.Method("POST"),
		h.Action(fmt.Sprintf("/note/%s/edit", n.ID)),
		h.Style("display:none"),
		g.Attr("data-show", fmt.Sprintf("$editNote === '%s'", n.ID)),
		h.Textarea(
			h.Name("body"),
			h.Rows("3"),
			h.Style("width:100%; font-family:monospace; font-size:16px; box-sizing:border-box"),
			g.Text(n.Text),
		),
		h.Div(h.Style("display:flex; gap:0.5em; margin-top:0.25em"),
			h.Button(h.Type("submit"), g.Text("save")),
			h.Button(h.Type("button"),
				g.Attr("data-on:click", "$editNote = ''"),
				g.Text("cancel"),
			),
		),
	)
}

func captureOverdueSection(noteList []note.Note) g.Node {
	if len(noteList) == 0 {
		return nil
	}
	return h.Div(
		h.Div(h.Style("padding: 0 1em; margin: 0.5em 0 0.25em; font-weight: bold"), g.Text("Overdue")),
		h.Div(h.Class("note-list"),
			g.Map(noteList, func(n note.Note) g.Node {
				return h.Div(h.Class("note-item"),
					h.Span(
						g.Attr("data-show", fmt.Sprintf("$editNote !== '%s'", n.ID)),
						h.Span(linkifyNode(n.Text)),
						noteActionsVisible(n, true),
					),
					inlineEditForm(n),
				)
			}),
		),
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
					h.Span(linkifyNode(n.Text)),
					scheduleButtons(n),
				)
			}),
		),
	)
}

const filterUntagged = "__untagged"

func tagHeader(tags []string, selected string) g.Node {
	pill := func(label, href string, active bool) g.Node {
		style := "border-radius:3px; padding:0.1em 0.4em; font-size:0.85em; color:#444; text-decoration:none; background:#eee"
		if active {
			style += "; outline:2px solid #888"
		}
		return h.A(h.Href(href), h.Style(style), g.Text(label))
	}
	return h.Div(
		h.ID("tag-header"),
		g.Attr("data-signals", fmt.Sprintf(`{"selectedTag":%q}`, selected)),
		h.Style("padding:0.5em 1em; display:flex; flex-wrap:wrap; gap:0.4em; border-bottom:1px solid #eee"),
		pill("all", "/capture/reference", selected == ""),
		pill("untagged", "/capture/reference?untagged=1", selected == filterUntagged),
		g.Map(tags, func(tag string) g.Node {
			href := "/capture/reference?tag=" + tag
			if tag == selected {
				href = "/capture/reference"
			}
			return pill("#"+tag, href, tag == selected)
		}),
	)
}

func (s *webservice) tagHeaderEl(owner, selected string) (g.Node, error) {
	tags, err := s.app.Notes.FindAllTagsForOwner(owner)
	if err != nil {
		return nil, err
	}
	return tagHeader(tags, selected), nil
}

func captureNoteList(noteList []note.Note, tagMap map[uuid.UUID][]string) g.Node {
	return h.Div(h.Class("note-list"),
		g.Map(noteList, func(n note.Note) g.Node {
			return captureNoteItem(n, tagMap[n.ID])
		}),
	)
}

// noteTagSignal returns the per-note signal name for the tag input.
func strSignal(sigs map[string]interface{}, key string) string {
	if v, ok := sigs[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func noteTagSignal(id uuid.UUID) string {
	return "nt_" + strings.ReplaceAll(id.String(), "-", "_")
}

func captureNoteItem(n note.Note, tags []string) g.Node {
	sig := noteTagSignal(n.ID)
	return h.Div(
		h.ID("capture-note-"+n.ID.String()),
		h.Class("note-item"),
		g.Attr("data-signals", fmt.Sprintf(`{"%s":""}`, sig)),
		h.Div(h.Style("display:flex; flex-wrap:wrap; gap:0.25em; align-items:baseline"),
			h.Span(linkifyNode(n.Text)),
			g.Map(tags, func(tag string) g.Node {
				return h.Button(
					h.Style("color:#555; background:#eee; border:1px solid #ccc; border-radius:3px; padding:0 0.4em; cursor:pointer; font-size:0.85em"),
					g.Attr("data-on:click", fmt.Sprintf("@post('/capture/notes/%s/tags/%s/remove')", n.ID, tag)),
					g.Text(tag+" ×"),
				)
			}),
			h.Form(
				g.Attr("data-on:submit", fmt.Sprintf("@post('/capture/notes/%s/tags')", n.ID)),
				h.Style("display:contents"),
				h.Input(
					g.Attr("data-bind", sig),
					h.Type("text"),
					h.Style("width:7em; font-size:0.85em"),
					h.Placeholder("add tag"),
				),
			),
		),
	)
}

func noteActions(n note.Note) g.Node {
	return noteActionsVisible(n, false)
}

func noteActionsVisible(n note.Note, visible bool) g.Node {
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
	editBtn := h.Button(
		h.Type("button"),
		h.Style("color:gray; padding:0"),
		g.Attr("data-on:click", fmt.Sprintf("$editNote = '%s'", n.ID)),
		g.Text("edit"),
	)
	if visible {
		return h.Span(
			h.Style("margin-left:0.5em"),
			g.If(includeDone, actionBtn("done", "done")),
			g.If(includeDone && includeReschedule, g.Text(" · ")),
			g.If(includeReschedule, actionBtn("reschedule", "reschedule")),
			g.Text(" · "),
			editBtn,
		)
	}
	return h.Span(
		g.Attr("data-class", fmt.Sprintf(`{"invisible": $activeNote !== '%s'}`, n.ID)),
		h.Class("invisible"),
		h.Style("margin-left:0.5em"),
		g.If(includeDone, actionBtn("done", "done")),
		g.If(includeDone && includeReschedule, g.Text(" · ")),
		g.If(includeReschedule, actionBtn("reschedule", "reschedule")),
		g.Text(" · "),
		editBtn,
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
					h.Span(
						g.Attr("data-show", fmt.Sprintf("$editNote !== '%s'", n.ID)),
						h.Span(g.Attr("data-on:click", fmt.Sprintf("$activeNote = $activeNote === '%s' ? '' : '%s'", n.ID, n.ID)), h.Style("cursor:pointer"), linkifyNode(n.Text)),
						noteActions(n),
					),
					inlineEditForm(n),
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
					h.Span(
						g.Attr("data-show", fmt.Sprintf("$editNote !== '%s'", n.ID)),
						h.Span(g.Attr("data-on:click", fmt.Sprintf("$activeNote = $activeNote === '%s' ? '' : '%s'", n.ID, n.ID)), h.Style("cursor:pointer"), linkifyNode(n.Text)),
						noteActions(n),
					),
					inlineEditForm(n),
				)
			}),
		),
	)
}

func captureTaskSection(heading string, noteList []note.Note, showStar bool) g.Node {
	if len(noteList) == 0 {
		return nil
	}
	return h.Div(
		h.Div(h.Style("padding: 0 1em; margin: 0.5em 0 0.25em; font-weight: bold"), g.Text(heading)),
		h.Div(h.Class("note-list"),
			g.Map(noteList, func(n note.Note) g.Node {
				return h.Div(h.Class("note-item"),
					h.Span(
						g.Attr("data-show", fmt.Sprintf("$editNote !== '%s'", n.ID)),
						g.If(showStar, starButton(n)),
						h.Span(g.Attr("data-on:click", fmt.Sprintf("$activeNote = $activeNote === '%s' ? '' : '%s'", n.ID, n.ID)), h.Style("cursor:pointer"), linkifyNode(n.Text)),
						noteActions(n),
					),
					inlineEditForm(n),
				)
			}),
		),
	)
}

func (s *webservice) postAddNoteTag(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sig := noteTagSignal(noteID)
	var sigs map[string]interface{}
	if err := datastar.ReadSignals(r, &sigs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tag := strings.TrimSpace(strSignal(sigs, sig))

	if tag != "" {
		_ = s.app.Commander.Send(commands.AddNoteTag{NoteID: noteID, Tag: tag}) // ignore duplicate
	}

	n, err := s.app.Notes.FindOne(noteID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	selectedTag := strSignal(sigs, "selectedTag")

	tags, _ := s.app.Notes.FindTagsForNote(noteID)
	header, err := s.tagHeaderEl(getUserInfo(r).Id, selectedTag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.MarshalAndPatchSignals(map[string]string{sig: ""})
	sse.PatchElementGostar(header)
	sse.PatchElementGostar(captureNoteItem(n, tags))
}

func (s *webservice) postRemoveNoteTag(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "noteID"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tag := chi.URLParam(r, "tag")
	_ = s.app.Commander.Send(commands.RemoveNoteTag{NoteID: noteID, Tag: tag})

	var sigs map[string]interface{}
	_ = datastar.ReadSignals(r, &sigs)
	selectedTag := strSignal(sigs, "selectedTag")

	n, err := s.app.Notes.FindOne(noteID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tags, _ := s.app.Notes.FindTagsForNote(noteID)
	header, err := s.tagHeaderEl(getUserInfo(r).Id, selectedTag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.PatchElementGostar(header)
	sse.PatchElementGostar(captureNoteItem(n, tags))
}

func (s *webservice) settingsIndex(w http.ResponseWriter, r *http.Request) {
	capturePage(g.Group{
		captureNavWithRequest(r, "/capture/tasks"),
		h.Div(h.Style("padding:1em"),
			h.A(h.Href("/logout"), g.Text("logout")),
		),
	}).Render(w)
}
