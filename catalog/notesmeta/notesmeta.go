package notesmeta

import (
	"slices"
	"time"
)

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
	Target       string
	DaysUntilDue func() int
}

type Subcategory struct {
	Slug        string
	DisplayName string
	Timeframes  []Timeframe
	Transitions []Transition
	DaysFn      func() int
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
	taskLater       = "later"
	taskDone        = "done"
)

type Timeframe struct {
	Slug        string
	EventName   string
	DisplayName string
	Days        func() int
}

var (
	Today     = Timeframe{Slug: "today", EventName: "today", DisplayName: "Today", Days: func() int { return 1 }}
	Tomorrow  = Timeframe{Slug: "tomorrow", EventName: "tommorow", DisplayName: "Tomorrow", Days: func() int { return 2 }}
	ThisWeek  = Timeframe{Slug: "thisweek", EventName: "thisweek", DisplayName: "ThisWeek", Days: func() int { return int(6 - time.Now().Weekday()) }}
	NextWeek  = Timeframe{Slug: "nextweek", EventName: "nextweek", DisplayName: "NextWeek", Days: func() int { return 7 + int(6-time.Now().Weekday()) }}
	ThisMonth = Timeframe{Slug: "thismonth", EventName: "thismonth", DisplayName: "ThisMonth", Days: func() int { return remainingDaysInMonth(time.Now(), 1) }}
	NextMonth = Timeframe{Slug: "nextmonth", EventName: "nextmonth", DisplayName: "NextMonth", Days: func() int { return remainingDaysInMonth(time.Now(), 2) }}
)

var Task = Category{
	Slug:        "task",
	DisplayName: "Tasks",
	Subcategories: SubcategoryList{
		{
			Slug:        taskUnscheduled,
			DisplayName: "Unscheduled",
			Transitions: []Transition{
				{
					Event:        Today.EventName,
					Target:       taskScheduled,
					DaysUntilDue: Today.Days,
				},
				{
					Event:        Tomorrow.EventName,
					Target:       taskScheduled,
					DaysUntilDue: Tomorrow.Days,
				},
				{
					Event:        ThisWeek.EventName,
					Target:       taskScheduled,
					DaysUntilDue: ThisWeek.Days,
				},
				{
					Event:        NextWeek.EventName,
					Target:       taskScheduled,
					DaysUntilDue: NextWeek.Days,
				},
				{
					Event:        ThisMonth.EventName,
					Target:       taskScheduled,
					DaysUntilDue: ThisMonth.Days,
				},
				{
					Event:        NextMonth.EventName,
					Target:       taskScheduled,
					DaysUntilDue: NextMonth.Days,
				},
				{
					Event:  "later",
					Target: taskLater,
				},
				{
					Event:  "done",
					Target: taskDone,
				},
			},
		},
		{
			Slug:        taskScheduled,
			DisplayName: "Scheduled",
			Timeframes:  []Timeframe{Today, Tomorrow, ThisWeek, NextWeek, ThisMonth, NextMonth},
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskLater,
			DisplayName: "Someday",
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskDone,
			DisplayName: "Done",
			Transitions: []Transition{
				{Event: "undo", Target: taskUnscheduled},
			},
		},
	},
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
	noteOther         = "other"
)

var Note = Category{
	Slug:        "reference",
	DisplayName: "Notes",
	Subcategories: SubcategoryList{
		{
			Slug:        noteUncategorized,
			DisplayName: "Uncategorized",
			Transitions: []Transition{
				{Event: noteBookmark, Target: noteBookmark},
				{Event: noteReference, Target: noteReference},
				{Event: noteGratitude, Target: noteGratitude},
				{Event: noteIdea, Target: noteIdea},
				{Event: noteObservation, Target: noteObservation},
				{Event: noteReflection, Target: noteReflection},
				{Event: noteRead, Target: noteRead},
				{Event: noteListen, Target: noteListen},
				{Event: noteWatch, Target: noteWatch},
				{Event: noteOther, Target: noteOther},
			},
		},
		{
			Slug:        noteBookmark,
			DisplayName: "Bookmark",
			Transitions: []Transition{
				{Event: "recategorize", Target: noteUncategorized},
			},
		},
		{
			Slug:        noteReference,
			DisplayName: "Remember",
			Transitions: []Transition{
				{Event: "recategorize", Target: noteUncategorized},
			},
		},
		{
			Slug:        noteGratitude,
			DisplayName: "Gratitude",
			Transitions: []Transition{
				{Event: "recategorize", Target: noteUncategorized},
			},
		},
		{
			Slug:        noteIdea,
			DisplayName: "Idea",
			Transitions: []Transition{
				{Event: "recategorize", Target: noteUncategorized},
			},
		},
		{
			Slug:        noteObservation,
			DisplayName: "Observation",
			Transitions: []Transition{
				{Event: "recategorize", Target: noteUncategorized},
			},
		},
		{
			Slug:        noteReflection,
			DisplayName: "Reflection",
			Transitions: []Transition{
				{Event: "recategorize", Target: noteUncategorized},
			},
		},
		{
			Slug:        noteRead,
			DisplayName: "Read",
			Transitions: []Transition{
				{Event: "recategorize", Target: noteUncategorized},
			},
		},
		{
			Slug:        noteListen,
			DisplayName: "Listen",
			Transitions: []Transition{
				{Event: "recategorize", Target: noteUncategorized},
			},
		},
		{
			Slug:        noteWatch,
			DisplayName: "Watch",
			Transitions: []Transition{
				{Event: "recategorize", Target: noteUncategorized},
			},
		},
		{
			Slug:        noteOther,
			DisplayName: "Other",
			Transitions: []Transition{
				{Event: "recategorize", Target: noteUncategorized},
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
