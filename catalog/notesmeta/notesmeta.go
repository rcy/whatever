package notesmeta

import (
	"fmt"
	"slices"
	"time"
)

var location = func() *time.Location {
	loc, err := time.LoadLocation("America/Creston")
	if err != nil {
		panic(err)
	}
	return loc
}()

type Category struct {
	Slug          string
	DisplayName   string
	Default       bool
	Subcategories SubcategoryList
}

func (c Category) DefaultSubcategory() Subcategory {
	if len(c.Subcategories) == 1 {
		return c.Subcategories[0]
	}
	return c.Subcategories[1]
}

func (c Category) Inbox() Subcategory {
	return c.Subcategories[0]
}

type Transition struct {
	Event        string
	TargetSlug   string
	DaysUntilDue func() int
}

type Subcategory struct {
	Slug        string
	DisplayName string
	Timeframes  []Timeframe
	Transitions TransitionList
}

type CategoryList []Category

func (l CategoryList) Get(slug string) Category {
	i := slices.IndexFunc(l, func(c Category) bool { return c.Slug == slug })
	if i == -1 {
		i = 0
	}
	return l[i]
}

type SubcategoryList []Subcategory

func (l SubcategoryList) Get(slug string) Subcategory {
	i := slices.IndexFunc(l, func(c Subcategory) bool { return c.Slug == slug })
	if i == -1 {
		i = 0
	}
	return l[i]
}

type TransitionList []Transition

// Return transition by event name
func (l TransitionList) Get(event string) (bool, Transition) {
	i := slices.IndexFunc(l, func(t Transition) bool { return t.Event == event })
	if i == -1 {
		return false, Transition{}
	}
	return true, l[i]
}

var Inbox = Category{
	Slug:        "inbox",
	DisplayName: "Inbox",
	Subcategories: SubcategoryList{
		{
			Slug: "default",
		},
	},
}

var DefaultCategory = Task

const (
	taskUnscheduled = "notnow"
	taskScheduled   = "scheduled"
	taskSomeday     = "someday"
	taskDone        = "done"
)

type Timeframe struct {
	Slug        string
	EventName   string
	DisplayName string
	Days        func() int
}

var TimeframeList = []Timeframe{
	Timeframe{Slug: "today", EventName: "today", DisplayName: "Today", Days: func() int { return 1 }},
	Timeframe{Slug: "tomorrow", EventName: "tommorow", DisplayName: "Tomorrow", Days: func() int { return 2 }},
	Timeframe{Slug: "thisweek", EventName: "thisweek", DisplayName: "ThisWeek", Days: func() int { return int(6 - time.Now().Weekday()) }},
	Timeframe{Slug: "nextweek", EventName: "nextweek", DisplayName: "NextWeek", Days: func() int { return 7 + int(6-time.Now().Weekday()) }},
	Timeframe{Slug: "thismonth", EventName: "thismonth", DisplayName: "ThisMonth", Days: func() int { return remainingDaysInMonth(time.Now(), 1) }},
	Timeframe{Slug: "nextmonth", EventName: "nextmonth", DisplayName: "NextMonth", Days: func() int { return remainingDaysInMonth(time.Now(), 2) }},
}

func timeframeLookup(slug string) (bool, Timeframe) {
	for _, tf := range TimeframeList {
		if tf.Slug == slug {
			return true, tf
		}
	}
	return false, Timeframe{}
}

// Return start time and end time of range (relative to now) for timerange given by slug
// Eg: TimeframeRange("tomorrow") returns midnight+1 day, midnight+2 days, nil
func TimeframeRange(slug string) (time.Time, time.Time, error) {
	for i, tf := range TimeframeList {
		if tf.Slug == slug {
			start := Midnight(time.Now().In(location))
			end := start.AddDate(0, 0, tf.Days())
			if i > 0 {
				start = start.AddDate(0, 0, TimeframeList[i-1].Days())
			} else {
				start = time.Unix(0, 0) // show overdue on today
			}
			return start, end, nil
		}
	}

	return time.Time{}, time.Time{}, fmt.Errorf("could not find timeframe")
}

var Task = Category{
	Slug:        "task",
	DisplayName: "Tasks",
	Subcategories: SubcategoryList{
		{
			Slug:        taskUnscheduled,
			DisplayName: "Unscheduled",
			Transitions: nil, // initialized in init() dynamically
		},
		{
			Slug:        taskScheduled,
			DisplayName: "Scheduled",
			Timeframes:  TimeframeList,
			Transitions: TransitionList{
				{Event: "reschedule", TargetSlug: taskUnscheduled},
				{Event: "done", TargetSlug: taskDone},
			},
		},
		{
			Slug:        taskSomeday,
			DisplayName: "Someday",
			Transitions: TransitionList{
				{Event: "reschedule", TargetSlug: taskUnscheduled},
				{Event: "done", TargetSlug: taskDone},
			},
		},
		{
			Slug:        taskDone,
			DisplayName: "Done",
			Transitions: TransitionList{
				{Event: "undo", TargetSlug: taskUnscheduled},
			},
		},
	},
}

func init() {
	txs := []Transition{}
	for _, tf := range TimeframeList {
		txs = append(txs, Transition{
			Event:        tf.EventName,
			TargetSlug:   taskScheduled,
			DaysUntilDue: tf.Days,
		})
	}
	txs = append(txs,
		Transition{
			Event:      "someday",
			TargetSlug: taskSomeday,
		},
		Transition{
			Event:      "done",
			TargetSlug: taskDone,
		},
	)
	Task.Subcategories[0].Transitions = txs
}

const (
	noteUncategorized = "process"
	noteReference     = "reference"
	noteBookmark      = "bookmark"
	noteGratitude     = "gratitude"
	noteIdea          = "idea"
	noteObservation   = "observation"
	noteReflection    = "reflection"
	noteRead          = "read"
	noteListen        = "listen"
	noteWatch         = "watch"
	noteQuote         = "quote"
	noteOther         = "other"
)

var Note = Category{
	Slug:        "reference",
	DisplayName: "Notes",
	Subcategories: SubcategoryList{
		{
			Slug:        noteUncategorized,
			DisplayName: "Uncategorized",
			Transitions: TransitionList{
				{Event: noteBookmark, TargetSlug: noteBookmark},
				{Event: noteReference, TargetSlug: noteReference},
				{Event: noteGratitude, TargetSlug: noteGratitude},
				{Event: noteIdea, TargetSlug: noteIdea},
				{Event: noteObservation, TargetSlug: noteObservation},
				{Event: noteReflection, TargetSlug: noteReflection},
				{Event: noteRead, TargetSlug: noteRead},
				{Event: noteListen, TargetSlug: noteListen},
				{Event: noteWatch, TargetSlug: noteWatch},
				{Event: noteQuote, TargetSlug: noteQuote},
				{Event: noteOther, TargetSlug: noteOther},
			},
		},
		{
			Slug:        noteBookmark,
			DisplayName: "Bookmark",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
		{
			Slug:        noteReference,
			DisplayName: "Remember",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
		{
			Slug:        noteGratitude,
			DisplayName: "Gratitude",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
		{
			Slug:        noteIdea,
			DisplayName: "Idea",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
		{
			Slug:        noteObservation,
			DisplayName: "Observation",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
		{
			Slug:        noteReflection,
			DisplayName: "Reflection",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
		{
			Slug:        noteRead,
			DisplayName: "Read",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
		{
			Slug:        noteListen,
			DisplayName: "Listen",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
		{
			Slug:        noteWatch,
			DisplayName: "Watch",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
		{
			Slug:        noteQuote,
			DisplayName: "Quote",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
		{
			Slug:        noteOther,
			DisplayName: "Other",
			Transitions: TransitionList{
				{Event: "recategorize", TargetSlug: noteUncategorized},
			},
		},
	},
}

var People = Category{
	Slug:        "people",
	DisplayName: "People",
	Subcategories: SubcategoryList{
		{
			Slug:        "default",
			DisplayName: "Default",
		},
	},
}

var Categories = CategoryList{Inbox, Task, Note, People}
var RefileCategories = CategoryList{Inbox, Task, Note}
