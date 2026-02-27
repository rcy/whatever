package web

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/kkdai/youtube/v2"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/catalog/notesmeta"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/projections/note"
	"github.com/starfederation/datastar-go/datastar"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

//go:embed style.css
var styles string

type Config struct {
	BaseURL            string
	GoogleClientID     string
	GoogleClientSecret string
	SessionSecret      string
}

type webservice struct {
	app         *app.App
	oauthConfig *oauth2.Config
	sessions    *sessionManager
	states      stateManager
	baseURL     string
}

func Server(app *app.App, cfg Config) (*chi.Mux, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("web: base URL is required for oauth redirect")
	}
	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		return nil, errors.New("web: google oauth client id and secret are required")
	}
	sessions, err := newSessionManager(cfg.SessionSecret)
	if err != nil {
		return nil, fmt.Errorf("web: %w", err)
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	svc := webservice{
		app: app,
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  baseURL + "/auth/callback",
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
				"openid",
			},
			Endpoint: google.Endpoint,
		},
		sessions: sessions,
		states:   stateManager{},
		baseURL:  baseURL,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/auth", svc.authHandler)
	r.Get("/auth/callback", svc.authCallbackHandler)
	r.Get("/logout", svc.logoutHandler)

	r.Group(func(r chi.Router) {
		r.Use(svc.authMiddleware)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/dsnotes/"+notesmeta.DefaultCategory.Slug, http.StatusSeeOther)
		})

		r.Get("/note/{id}", svc.showNote)

		r.Get("/events", svc.eventsIndex)

		r.Get("/dsnotes/{category}", svc.notesIndexRedirect)
		r.Get("/dsnotes/{category}/{subcategory}", svc.notesIndex)
		r.Get("/dsnotes/{category}/{subcategory}/{timeframe}", svc.notesIndex)

		r.Get("/dsnotes/people", svc.notesPeople)
		r.Get("/dsnotes/people/{handle}", svc.notesPeople)

		r.Post("/dsnotes", svc.postNotesHandler)
		r.Post("/refile/{noteID}/{category}", svc.postRefileNote)
		r.Post("/trans/{noteID}/{event}", svc.postSubcategoryTransition)
		r.Post("/delete/{noteID}", svc.postDeleteNote)
		r.Post("/undelete/{noteID}", svc.postUndeleteNote)
	})

	return r, nil
}

type signals struct {
	Body            string `json:"body"`
	ViewCategory    string `json:"viewCategory"`
	ViewSubcategory string `json:"viewSubcategory"`
}

func (s *webservice) postNotesHandler(w http.ResponseWriter, r *http.Request) {
	userInfo := getUserInfo(r)

	var signals signals
	err := datastar.ReadSignals(r, &signals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if signals.Body != "" {
		err := s.app.Commander.Send(commands.CreateNote{
			Owner:       userInfo.Id,
			NoteID:      uuid.New(),
			Text:        signals.Body,
			Category:    notesmeta.Inbox.Slug,
			Subcategory: notesmeta.Inbox.Inbox().Slug,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	sse := datastar.NewSSE(w, r)

	// if we are looking at the inbox, reload the page to immediately show the item that was added to the inbox
	if signals.ViewCategory == "inbox" {
		sse.Redirect("")
		return
	}

	// ...otherwise, patch up the ui

	signals.Body = ""
	sse.MarshalAndPatchSignals(signals)

	inbox, err := s.inboxHeader(userInfo.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse.PatchElementGostar(inbox)
}

// Wrap ui header element with data fetching
func (s *webservice) header(r *http.Request, viewCategory string, viewSubcategory string) (g.Node, error) {
	owner := getUserInfo(r)

	categoryCounts, err := s.app.Notes.CategoryCounts(owner.Id)
	if err != nil {
		return nil, fmt.Errorf("Notes.CategoryCounts: %w", err)
	}
	subcategoryCounts, err := s.app.Notes.SubcategoryCounts(owner.Id, viewCategory)
	if err != nil {
		return nil, fmt.Errorf("Notes.SubcategoryCounts: %w", err)
	}

	inbox, err := s.inboxHeader(owner.Id)
	if err != nil {
		return nil, fmt.Errorf("inboxHeader: %w", err)
	}

	category := notesmeta.Categories.Get(viewCategory)
	inboxNoteList, err := s.app.Notes.FindAllByCategoryAndSubcategory(owner.Id, category.Slug, category.Inbox().Slug)
	if err != nil {
		return nil, fmt.Errorf("FindAllByCategoryAndSubcategory: %w", err)
	}
	slices.Reverse(inboxNoteList)

	return h.Div(
		inbox,
		greenHeader(viewCategory, viewSubcategory, categoryCounts, subcategoryCounts),
		h.Div(h.Style("background: #32cd3260; padding: 0 5px"), notes(inboxNoteList)),
		pinkHeader(viewCategory, viewSubcategory, categoryCounts, subcategoryCounts)), nil
}

func (s *webservice) inboxHeader(owner string) (g.Node, error) {
	noteList, err := s.app.Notes.FindAllByCategory(owner, notesmeta.Inbox.Slug)
	if err != nil {
		return nil, fmt.Errorf("FindAllByCategory: %w", err)
	}
	slices.Reverse(noteList)

	return h.Div(h.ID("inbox"), h.Style("background:#eeeeee"),
		h.Div(h.Style("display:flex; gap:5px; padding:5px"),
			h.Div(h.Style("font-weight:bold"), g.Text("NOTNOW //")),
			h.Div(h.Style("flex:1"), inboxInput()),
		),
		h.Div(h.Style("padding: 0 5px"), notes(noteList)),
	), nil
}

func (s *webservice) eventsIndex(w http.ResponseWriter, r *http.Request) {
	events, err := s.app.EventDebugger.DebugEvents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(events)

	h.HTML(
		h.Body(
			h.H1(g.Text("events")),
			g.Map(events, func(ev evoke.RecordedEvent) g.Node {
				data, err := json.Marshal(ev)
				if err != nil {
					return h.Div(h.Style("color: red"), g.Text(err.Error()))
				}
				return h.Div(g.Text(string(data)))
			}),
		),
	).Render(w)
}

func (s *webservice) showNote(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "id")
	note, err := s.app.Notes.FindOne(noteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// show the note
	// show some buttons
	// edit / delete / delete

	links, err := noteLinksEl(note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	actions := h.Div(
		h.Div(h.A(g.Text("torrent"), h.Href("https://thepiratebay11.com/search/"+url.PathEscape(note.Text)))),
		h.Div(h.A(g.Text("ddg"), h.Href("https://duckduckgo.com/?q="+url.QueryEscape(note.Text)))),
		h.Div(h.A(g.Text("goog"), h.Href("https://www.google.com/search?q="+url.QueryEscape(note.Text)))),
		h.Div(h.A(g.Text("yt"), h.Href("https://www.youtube.com/results?search_query="+url.PathEscape(note.Text)))),
		h.Div(h.A(g.Text("wiki"), h.Href("https://en.wikipedia.org/w/index.php?title=Special:Search&search="+url.QueryEscape(note.Text)))),
	)

	page := h.Div(
		noteEl(note),
		// h.Form(
		// 	g.Attr("data-on:submit", fmt.Sprintf("@post('/note/%s/comment', {contentType: 'form'})", note.ID)),
		// 	h.Textarea(
		// 		h.Name("bodyx"),
		// 		h.Rows("3"),
		// 		h.Style("width:100%"),
		// 	),
		// 	h.Button(g.Text("submit")),
		// ),
		links,
		actions,
		//youtubeDownloadButton(note),
	)

	content, err := s.page(r, note.Category, note.Subcategory, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	content.Render(w)
}

func noteLinksEl(note note.Note) (g.Node, error) {
	links := getLinks(note.Text)

	return h.Div(
		g.Map(links, func(link string) g.Node {
			if strings.Contains(link, "youtu") {
				embed, err := youtubeEmbed(link)
				if err != nil {
					return h.Div(g.Text("error: " + err.Error()))
				}

				return h.Div(embed)
			} else {
				return h.Div(g.Text(link))
			}
		})), nil
}

func youtubeEmbed(link string) (g.Node, error) {
	videoID, err := youtube.ExtractVideoID(link)
	if err != nil {
		return nil, err
	}
	return h.IFrame(h.Width("560"), h.Height("315"),
		h.Src("https://www.youtube.com/embed/"+videoID),
		g.Attr("allowfullscreen")), nil
}

// redirect to the default subcategory
func (s *webservice) notesIndexRedirect(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	defaultSubcategory := notesmeta.Categories.Get(category).DefaultSubcategory().Slug
	http.Redirect(w, r, fmt.Sprintf("%s/%s", category, defaultSubcategory), http.StatusSeeOther)
}

func (s *webservice) notesIndex(w http.ResponseWriter, r *http.Request) {
	categoryParam := chi.URLParam(r, "category")
	subcategoryParam := chi.URLParam(r, "subcategory")
	owner := getUserInfo(r)

	var noteList []note.Note
	var err error

	if subcategoryParam == "all" {
		noteList, err = s.app.Notes.FindAllByCategory(owner.Id, categoryParam)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		noteList, err = s.app.Notes.FindAllByCategoryAndSubcategory(owner.Id, categoryParam, subcategoryParam)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	slices.Reverse(noteList)

	timeframe := chi.URLParam(r, "timeframe")
	if timeframe != "" {
		start, end, err := notesmeta.TimeframeRange(timeframe)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// filter out notes not within timeframe
		filteredNotes := []note.Note{}
		for _, note := range noteList {
			if note.Due != nil {
				due := time.Unix(*note.Due, 0)
				if due.After(start) && (due.Before(end) || due.Equal(end)) {
					filteredNotes = append(filteredNotes, note)
				}
			}
		}
		noteList = filteredNotes
	}

	content, err := s.page(r, categoryParam, subcategoryParam, h.Div(
		notes(noteList),
	))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	content.Render(w)
}

func (s *webservice) notesPeople(w http.ResponseWriter, r *http.Request) {
	owner := getUserInfo(r)
	handleParam := chi.URLParam(r, "handle")

	people, err := s.app.Notes.FindAllPeople(owner.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var notes []note.Note
	if handleParam == "" {
		notes, err = s.app.Notes.FindAllWithMention(owner.Id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		notes, err = s.app.Notes.FindAllByPerson(owner.Id, handleParam)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	content, err := s.page(r, "people", "", h.Div(
		h.Div(h.Style("background: pink; padding: 5px; display:flex; justify-content: space-between;"),
			h.Div(h.Style("display: flex; gap: 5px;"),
				g.Map(people, func(handle string) g.Node {
					text := fmt.Sprintf("%s", g.Text(handle))
					var style g.Node
					if handle == handleParam {
						style = h.Style("font-weight: bold")
					}
					return h.Div(h.A(style, g.Text(text), h.Href(fmt.Sprintf("/dsnotes/people/%s", handle))))
				}),
			),
			h.Div(h.A(g.Text("all"), h.Href(fmt.Sprintf("/dsnotes/people")))),
		),
		g.Map(notes, func(note note.Note) g.Node {
			return noteEl(note)
		})))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	content.Render(w)
}

func (s *webservice) page(r *http.Request, category string, subcategory string, node g.Node) (g.Node, error) {
	headerEl, err := s.header(r, category, subcategory)
	if err != nil {
		return nil, err
	}
	return h.HTML(
		h.Head(
			h.Script(h.Type("module"), h.Src("https://cdn.jsdelivr.net/gh/starfederation/datastar@1.0.0-RC.7/bundles/datastar.js")),
			h.StyleEl(g.Raw(styles)),
		),
		h.Body(
			h.Div(g.Attr("data-signals", fmt.Sprintf("{viewCategory: '%s', viewSubcategory: '%s'}", category, subcategory))),
			h.Div(h.Style("display:flex;flex-direction:column;gap:10px"),
				h.Div(headerEl),
				h.Div(node),
			),
		),
	), nil
}

func (s *webservice) postRefileNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noteID")
	categoryName := chi.URLParam(r, "category")

	var signals signals
	err := datastar.ReadSignals(r, &signals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	noteID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.app.Commander.Send(commands.ClearNoteDue{NoteID: noteID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.app.Commander.Send(commands.SetNoteCategory{NoteID: noteID, Category: categoryName})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	headerEl, err := s.header(r, signals.ViewCategory, signals.ViewSubcategory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.PatchElementGostar(headerEl)

	note, err := s.app.Notes.FindOne(noteID.String())
	if err != nil {
		sse.ConsoleError(err)
		return
	}
	noteEl := noteEl(note)

	sse.PatchElementGostar(noteEl)
}

func (s *webservice) postSubcategoryTransition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noteID")
	event := chi.URLParam(r, "event")

	var signals signals
	err := datastar.ReadSignals(r, &signals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(w, r)

	noteID, err := uuid.Parse(id)
	if err != nil {
		sse.ConsoleError(err)
		return
	}

	note, err := s.app.Notes.FindOne(id)
	if err != nil {
		sse.ConsoleError(err)
		return
	}

	err = s.app.Commander.Send(commands.TransitionNoteSubcategory{NoteID: note.ID, TransitionEvent: event})
	if err != nil {
		sse.ConsoleError(err)
		return
	}

	headerEl, err := s.header(r, signals.ViewCategory, signals.ViewSubcategory)
	if err != nil {
		sse.ConsoleError(err)
		return
	}

	note, err = s.app.Notes.FindOne(noteID.String())
	if err != nil {
		sse.ConsoleError(err)
		return
	}
	noteEl := noteEl(note)

	sse.PatchElementGostar(noteEl)
}

func (s *webservice) postDeleteNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noteID")

	noteID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	note, err := s.app.Notes.FindOne(noteID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.app.Commander.Send(commands.DeleteNote{NoteID: noteID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var signals signals
	err = datastar.ReadSignals(r, &signals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	headerEl, err := s.header(r, signals.ViewCategory, signals.ViewSubcategory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.PatchElementGostar(headerEl)

	sse.PatchElementGostar(deletedNoteEl(note))
}

func (s *webservice) postUndeleteNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noteID")

	noteID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.app.Commander.Send(commands.UndeleteNote{NoteID: noteID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var signals signals
	err = datastar.ReadSignals(r, &signals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	headerEl, err := s.header(r, signals.ViewCategory, signals.ViewSubcategory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.PatchElementGostar(headerEl)

	note, err := s.app.Notes.FindOne(noteID.String())
	if err != nil {
		sse.ConsoleError(err)
		return
	}
	sse.PatchElementGostar(noteEl(note))
}

func inboxInput() g.Node {
	return h.Form(h.ID("input-form"), g.Attr("data-on:submit", "@post('/dsnotes')"), h.Style("margin:0"),
		h.Input(
			g.Attr("data-bind", "body"),
			h.Style("width:100%"),
			h.Placeholder("Add a note to inbox..."),
			h.AutoFocus(),
		),
	)
}

func greenHeader(category string, subcategory string, categoryCounts []note.CategoryCount, subcategoryCounts []note.SubcategoryCount) g.Node {
	return h.Div(h.Style("background: lime; padding: 5px; display:flex; gap: 20px"),
		h.Div(h.Style("display:flex; gap:5px"),
			h.Div(h.Style("display: flex; gap: 5px"),
				g.Map(notesmeta.Categories, func(c notesmeta.Category) g.Node {
					if c.Slug == notesmeta.Inbox.Slug {
						return nil
					}
					text := fmt.Sprintf("%s", c.DisplayName)
					if c.Slug == category {
						return h.Div(
							h.A(h.Style("font-weight: bold"),
								g.Text(text),
								h.Href("/dsnotes/"+c.Slug)))
					} else {
						return h.Div(h.A(g.Text(text), h.Href("/dsnotes/"+c.Slug)))
					}
				}))))
}

func pinkHeader(categorySlug string, subcategory string, categoryCounts []note.CategoryCount, subcategoryCounts []note.SubcategoryCount) g.Node {
	category := notesmeta.Categories.Get(categorySlug)
	return g.If(len(category.Subcategories) > 1,
		h.Div(h.Style("background: pink; padding: 5px; display:flex; justify-content: space-between;"),
			h.Div(h.Style("display: flex; gap: 5px"),
				g.Map(category.Subcategories, func(sub notesmeta.Subcategory) g.Node {
					if sub.Slug == category.Inbox().Slug {
						return nil
					}
					if len(sub.Timeframes) > 0 {
						return g.Group{g.Map(sub.Timeframes, func(tf notesmeta.Timeframe) g.Node {
							style := h.Style("")
							return h.Div(h.A(style, g.Text(tf.DisplayName), h.Href(fmt.Sprintf("/dsnotes/%s/%s/%s", categorySlug, sub.Slug, tf.Slug))))
						})}
					} else {
						text := fmt.Sprintf("%s", g.Text(sub.DisplayName))
						var style g.Node
						if sub.Slug == subcategory {
							style = h.Style("font-weight: bold")
						}
						return h.Div(h.A(style, g.Text(text), h.Href(fmt.Sprintf("/dsnotes/%s/%s", categorySlug, sub.Slug))))
					}
				}),
			),
			h.Div(h.A(g.Text("all"), h.Href(fmt.Sprintf("/dsnotes/%s/all", categorySlug)))),
		))
}

func notes(noteList []note.Note) g.Node {
	return h.Div(h.Style("display:flex; flex-direction:column; gap:10px; margin-bottom:1em"),
		g.Map(noteList, func(note note.Note) g.Node {
			return noteEl(note)
		}),
	)
}

func noteID(note note.Note) string {
	return fmt.Sprintf("note-%s", note.ID)
}

// Return a link to the note
func noteLink(note note.Note) string {
	return fmt.Sprintf("/note/%s", note.ID)
}

func noteEl(note note.Note) g.Node {
	return h.Div(h.ID(noteID(note)),
		h.Div(
			h.A(h.Href(noteLink(note)),
				h.Span(
					h.Span(h.Style("color:gray"), g.Text(noteCategoryDisplay(note))),
					h.Span(g.Raw("&nbsp;")),
					h.Span(linkifyNode(note.Text)),
					g.Iff(note.Due != nil, func() g.Node {
						until := int(math.Ceil(float64(time.Until(time.Unix(*note.Due, 0)))/float64(24*time.Hour) - 1))
						return h.Span(g.Text(fmt.Sprintf(" %dd", until)))
					}),
				)),
		),
		h.Div(h.Style("color: gray; font-size: 70%; margin-top: -3px;"),
			h.Div(h.Style("display:flex; gap:2px"),
				refile(note),
			),
		),
	)
}

func noteCategoryDisplay(n note.Note) string {
	cat := notesmeta.Categories.Get(n.Category)
	subcat := cat.Subcategories.Get(n.Subcategory)

	// don't print the subcategory if the category doesn't have more than 1
	if len(cat.Subcategories) <= 1 {
		return cat.DisplayName
	}
	return subcat.DisplayName // + " " + cat.DisplayName
}

func deletedNoteEl(note note.Note) g.Node {
	return h.Div(h.ID(noteID(note)),
		h.Div(h.Style("color: gray; text-decoration: line-through"),
			h.A(h.Href(noteLink(note)),
				linkifyNode(note.Status+" "+note.Text)),
		),
		h.Div(h.Style("color: gray; font-size: 70%; margin-top: -3px"),
			h.Div(h.Style("display:flex; gap:2px"),
				undeleteButton(note.ID),
			),
		),
	)
}

func refile(note note.Note) g.Node {
	if note.Category == "inbox" {
		return h.Div(h.Style("display:flex; gap:2px"),
			g.Map(notesmeta.RefileCategories, func(c notesmeta.Category) g.Node {
				if c.Slug == notesmeta.Inbox.Slug {
					return nil
				}
				return refileButton(note.ID, c.Slug, c.DisplayName)
			}),
			deleteButton(note.ID),
		)
	}

	transitions := notesmeta.Categories.Get(note.Category).Subcategories.Get(note.Subcategory).Transitions

	return h.Div(h.Style("display:flex; gap:2px"),
		g.Group{
			g.Map(transitions,
				func(t notesmeta.Transition) g.Node {
					return subfileButton(note.ID, t)
				},
			),
			g.If(len(transitions) > 0, g.Text(" | ")),
			refileButton(note.ID, "inbox", "refile"),
			deleteButton(note.ID),
		},
	)
}

func refileButton(noteID uuid.UUID, category string, label string) g.Node {
	url := fmt.Sprintf("/refile/%s/%s", noteID, category)
	return h.Button(h.Class("link"), g.Attr("data-on:click", fmt.Sprintf("@post('%s')", url)), g.Text(label))
}

func subfileButton(noteID uuid.UUID, t notesmeta.Transition) g.Node {
	url := fmt.Sprintf("/trans/%s/%s", noteID, t.Event)
	return h.Button(h.Class("link"), g.Attr("data-on:click", fmt.Sprintf("@post('%s')", url)), g.Text(t.Event))
}

func deleteButton(noteID uuid.UUID) g.Node {
	url := fmt.Sprintf("/delete/%s", noteID)
	return h.Button(
		h.Class("link"),
		g.Attr("data-on:click", fmt.Sprintf("@post('%s')", url)),
		g.Text("delete"))
}

func undeleteButton(noteID uuid.UUID) g.Node {
	url := fmt.Sprintf("/undelete/%s", noteID)
	return h.Button(
		h.Class("link"),
		g.Attr("data-on:click", fmt.Sprintf("@post('%s')", url)),
		g.Text("undelete"))
}
