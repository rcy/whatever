package notesmeta

import (
	"slices"
)

type Category struct {
	Name          string
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
	Name        string
	Transitions []Transition
}

type CategoryList []Category

func (l CategoryList) Get(name string) Category {
	i := slices.IndexFunc(l, func(c Category) bool { return c.Name == name })
	if i == -1 {
		i = 0
	}
	return l[i]
}

type SubcategoryList []Subcategory

func (l SubcategoryList) Get(name string) Subcategory {
	i := slices.IndexFunc(l, func(c Subcategory) bool { return c.Name == name })
	if i == -1 {
		i = 0
	}
	return l[i]
}

var Inbox = Category{
	Name: "inbox",
	Subcategories: SubcategoryList{
		{
			Name: "default",
		},
	},
}

var DefaultCategory = Inbox

const (
	taskNext    = "next"
	taskStarted = "started"
	taskNotnow  = "notnow"
	taskDone    = "done"
)

var Task = Category{
	Name: "task",
	Subcategories: SubcategoryList{
		{
			Name: taskNext,
			Transitions: []Transition{
				{Event: "start", Target: taskStarted},
				{Event: "notnow", Target: taskNotnow},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Name: "started",
			Transitions: []Transition{
				{Event: "pause", Target: taskNext},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Name: "notnow",
			Transitions: []Transition{
				{Event: "ready", Target: taskNext},
			},
		},
		{
			Name: "done",
			Transitions: []Transition{
				{Event: "undo", Target: taskNext},
			},
		},
	},
}

const (
	referenceProcess = "process"
	referenceArchive = "archive"
	referenceRead    = "read"
)

var Reference = Category{
	Name: "reference",
	Subcategories: SubcategoryList{
		{
			Name: referenceProcess,
			Transitions: []Transition{
				{Event: "archive", Target: referenceArchive},
			},
		},
		{
			Name: referenceArchive,
			Transitions: []Transition{
				{Event: "unarchive", Target: referenceRead},
			},
		},
	},
}

var Categories = CategoryList{Inbox, Task, Reference}
