package web

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rcy/whatever/commands/service"
	"github.com/rcy/whatever/models"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type ServeCmd struct {
	Port string `default:"9999"`
}

func (c *ServeCmd) Run(s *service.Service) error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	svc := webservice{Service: *s}
	r.Get("/", svc.index)
	fmt.Println("listening on port", c.Port)
	return http.ListenAndServe(":"+c.Port, r)
}

type webservice struct {
	service.Service
}

func (s *webservice) index(w http.ResponseWriter, r *http.Request) {
	var events []models.Event
	err := s.DB.Select(&events, `select * from events order by event_id desc`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.HTML(
		h.Body(
			h.H1(g.Text("whatever events")),
			h.Table(
				h.Body(
					g.Map(events, func(event models.Event) g.Node {
						return h.Tr(
							h.Td(h.A(h.Code(g.Text(event.AggregateID[0:7])))),
							h.Td(g.Text(event.EventType)),
							h.Td(h.Code(g.Text(string(event.EventData)))),
						)
					}),
				),
			),
		),
	).Render(w)
}
