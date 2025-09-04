package reminders

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/en"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/ids"
	"github.com/rcy/whatever/models"
)

type Cmd struct {
	Ls ListCmd `cmd:"" default:"withargs"`
	//Show     ShowCmd     `cmd:""`
	Add      AddCmd      `cmd:""`
	Rm       DeleteCmd   `cmd:""`
	Undelete UndeleteCmd `cmd:""`
}

type ListCmd struct {
	Deleted bool `help:"Show deleted reminders"`
}

func (c *ListCmd) Run(app *app.Service) error {
	type id string
	type reminder struct {
		text    string
		when    time.Time
		deleted bool
	}
	reminders := make(map[id]reminder)
	var events []models.Event
	err := app.ES.DBTodo.Select(&events, `select * from events where aggregate_type = 'reminder' order by event_id asc`)
	if err != nil {
		return fmt.Errorf("Select events: %w", err)
	}
	for _, event := range events {
		switch event.EventType {
		case "ReminderCreated":
			payload := struct {
				Text string
				When time.Time
			}{}
			err := json.Unmarshal(event.EventData, &payload)
			if err != nil {
				return fmt.Errorf("Unmarshal: %w", err)
			}
			reminders[id(event.AggregateID)] = reminder{text: payload.Text, when: payload.When}
		case "ReminderDeleted":
			reminder, ok := reminders[id(event.AggregateID)]
			if ok {
				reminder.deleted = true
				reminders[id(event.AggregateID)] = reminder
			}
		case "ReminderUndeleted":
			reminder, ok := reminders[id(event.AggregateID)]
			if ok {
				reminder.deleted = false
				reminders[id(event.AggregateID)] = reminder
			}
		default:
			return fmt.Errorf("unhandled event.EventType: %s", event.EventType)
		}
	}

	for id, reminder := range reminders {
		if c.Deleted && reminder.deleted || !c.Deleted && !reminder.deleted {
			since := -time.Since(reminder.when).Round(time.Second)

			fmt.Printf("%s %s %s\n", id[0:7], since, reminder.text)
		}
	}

	return nil
}

type AddCmd struct {
	Input []string `arg:""`
}

func (c *AddCmd) Run(app *app.Service) error {
	input := strings.Join(c.Input, " ")

	when, text, err := parseTimeAndTask(input)
	if err != nil {
		return fmt.Errorf("parseTimeAndTask: %w", err)
	}

	payload := struct {
		Text  string
		When  time.Time
		Input string
	}{
		Text:  text,
		When:  when,
		Input: input,
	}

	err = app.ES.InsertEvent("ReminderCreated", "reminder", ids.New(), payload)
	if err != nil {
		return fmt.Errorf("insertEvent: %w", err)
	}

	return nil
}

func parseTimeAndTask(input string) (time.Time, string, error) {
	w := when.New(nil)
	w.Add(en.All...) // add all English rules

	now := time.Now()
	result, err := w.Parse(input, now)
	if err != nil {
		return time.Time{}, "", err
	}
	if result == nil {
		return time.Time{}, input, nil // couldn't parse a time
	}

	// remove the time expression from the input
	remaining := strings.TrimSpace(strings.Replace(input, result.Text, "", 1))

	return result.Time, remaining, nil
}

type DeleteCmd struct {
	ID string `arg:""`
}

func (c *DeleteCmd) Run(app *app.Service) error {
	aggID, err := app.ES.GetAggregateID(c.ID)
	if err != nil {
		return err
	}

	_, err = app.ES.DBTodo.Exec(`insert into events(aggregate_id, aggregate_type, event_type, event_data) values (?,?,?,?)`, aggID, "reminder", "ReminderDeleted", "{}")
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}

type UndeleteCmd struct {
	ID string `arg:""`
}

func (c *UndeleteCmd) Run(app *app.Service) error {
	aggID, err := app.ES.GetAggregateID(c.ID)
	if err != nil {
		return err
	}

	err = app.ES.InsertEvent("ReminderUndeleted", "reminder", aggID, nil)
	if err != nil {
		return err
	}

	return nil
}
