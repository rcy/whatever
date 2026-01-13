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

var Task = Category{
	Name: "task",
	Subcategories: SubcategoryList{
		{
			Name: "ready",
			Transitions: []Transition{
				{Event: "start", Target: "working"},
				{Event: "notnow", Target: "notnow"},
				{Event: "done", Target: "done"},
			},
		},
		{
			Name: "working",
			Transitions: []Transition{
				{Event: "pause", Target: "ready"},
				{Event: "done", Target: "done"},
			},
		},
		{
			Name: "notnow",
			Transitions: []Transition{
				{Event: "ready", Target: "ready"},
			},
		},
		{
			Name: "done",
			Transitions: []Transition{
				{Event: "undo", Target: "ready"},
			},
		},
	},
}

var Reference = Category{
	Name: "reference",
	Subcategories: SubcategoryList{
		{
			Name: "process",
			Transitions: []Transition{
				{Event: "archive", Target: "archive"},
			},
		},
		{
			Name: "archive",
			Transitions: []Transition{
				{Event: "unarchive", Target: "read"},
			},
		},
	},
}

var Categories = CategoryList{Inbox, Task, Reference}
