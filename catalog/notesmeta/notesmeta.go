package notesmeta

import (
	"slices"
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
	taskNext        = "next"
	taskUnscheduled = "notnow"
	taskThisWeek    = "thisweek"
	taskThisMonth   = "thismonth"
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
				{Event: "today", Target: taskNext},
				{Event: "week", Target: taskThisWeek},
				{Event: "month", Target: taskThisMonth},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskNext,
			DisplayName: "Today",
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskThisWeek,
			DisplayName: "Week",
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskThisMonth,
			DisplayName: "Month",
			Transitions: []Transition{
				{Event: "reschedule", Target: taskUnscheduled},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskDone,
			DisplayName: "Done!",
			Transitions: []Transition{
				{Event: "undo", Target: taskNext},
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
