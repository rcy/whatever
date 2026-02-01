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

var DefaultCategory = Inbox

const (
	taskNext      = "next"
	taskNotnow    = "notnow"
	taskThisWeek  = "thisweek"
	taskThisMonth = "thismonth"
	taskDone      = "done"
)

var Task = Category{
	Slug:        "task",
	DisplayName: "Task",
	Subcategories: SubcategoryList{
		{
			Slug:        taskNotnow,
			DisplayName: "Unscheduled",
			Transitions: []Transition{
				{Event: "today", Target: taskNext},
				{Event: "thisweek", Target: taskThisWeek},
				{Event: "thismonth", Target: taskThisMonth},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskNext,
			DisplayName: "Today",
			Transitions: []Transition{
				{Event: "reschedule", Target: taskNotnow},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskThisWeek,
			DisplayName: "This Week",
			Transitions: []Transition{
				{Event: "reschedule", Target: taskNotnow},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Slug:        taskThisMonth,
			DisplayName: "This Month",
			Transitions: []Transition{
				{Event: "reschedule", Target: taskNotnow},
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
	referenceProcess = "process"
)

var Reference = Category{
	Slug:        "reference",
	DisplayName: "Note",
	Subcategories: SubcategoryList{
		{
			Slug:        referenceProcess,
			DisplayName: "Process",
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

var Categories = CategoryList{Inbox, Task, Reference, People}
var RefileCategories = CategoryList{Inbox, Task, Reference}
