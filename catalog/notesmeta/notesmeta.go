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
			Transitions: TransitionList{
				{
					Event:        Today.EventName,
					TargetSlug:   taskScheduled,
					DaysUntilDue: Today.Days,
				},
				{
					Event:        Tomorrow.EventName,
					TargetSlug:   taskScheduled,
					DaysUntilDue: Tomorrow.Days,
				},
				{
					Event:        ThisWeek.EventName,
					TargetSlug:   taskScheduled,
					DaysUntilDue: ThisWeek.Days,
				},
				{
					Event:        NextWeek.EventName,
					TargetSlug:   taskScheduled,
					DaysUntilDue: NextWeek.Days,
				},
				{
					Event:        ThisMonth.EventName,
					TargetSlug:   taskScheduled,
					DaysUntilDue: ThisMonth.Days,
				},
				{
					Event:        NextMonth.EventName,
					TargetSlug:   taskScheduled,
					DaysUntilDue: NextMonth.Days,
				},
				{
					Event:      "later",
					TargetSlug: taskLater,
				},
				{
					Event:      "done",
					TargetSlug: taskDone,
				},
			},
		},
		{
			Slug:        taskScheduled,
			DisplayName: "Scheduled",
			Timeframes:  []Timeframe{Today, Tomorrow, ThisWeek, NextWeek, ThisMonth, NextMonth},
			Transitions: TransitionList{
				{Event: "reschedule", TargetSlug: taskUnscheduled},
				{Event: "done", TargetSlug: taskDone},
			},
		},
		{
			Slug:        taskLater,
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
