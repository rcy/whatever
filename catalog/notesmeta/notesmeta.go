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
	Event  string
	Target string
}

type Subcategory struct {
	Slug        string
	DisplayName string
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
	taskToday       = "next"
	taskTomorrow    = "tomorrow"
	taskUnscheduled = "notnow"
	taskThisWeek    = "thisweek"
	taskNextWeek    = "nextweek"
	taskThisMonth   = "thismonth"
	taskNextMonth   = "nextmonth"
	taskDone        = "done"
)

var Task = Category{
	Slug:        "task",
	DisplayName: "Tasks",
	Subcategories: SubcategoryList{
		{
			Slug:        taskUnscheduled,
			DisplayName: "Unscheduled",
			Transitions: []Transition{
				{Event: "today", Target: taskToday},
				{Event: "tommorow", Target: taskTomorrow},
				{Event: "thisweek", Target: taskThisWeek},
				{Event: "nextweek", Target: taskNextWeek},
				{Event: "thismonth", Target: taskThisMonth},
				{Event: "nextmonth", Target: taskNextMonth},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskToday,
			DisplayName: "Today",
			DaysFn:      func() int { return 1 },
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskTomorrow,
			DisplayName: "Tomorrow",
			DaysFn:      func() int { return 2 },
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskThisWeek,
			DisplayName: "ThisWeek",
			DaysFn:      func() int { return int(7 - time.Now().Weekday()) },
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskNextWeek,
			DisplayName: "NextWeek",
			DaysFn:      func() int { return 7 + int(7-time.Now().Weekday()) },
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskThisMonth,
			DisplayName: "ThisMonth",
			DaysFn: func() int {
				return remainingDaysInMonth(time.Now(), 1)
			},
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskNextMonth,
			DisplayName: "NextMonth",
			DaysFn: func() int {
				return remainingDaysInMonth(time.Now(), 2)
			},
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskDone,
			DisplayName: "Done",
			Transitions: []Transition{
				{Event: "undo", Target: taskToday},
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
