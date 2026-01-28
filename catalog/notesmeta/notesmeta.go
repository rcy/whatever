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
	taskNext   = "next"
	taskNotnow = "notnow"
	taskDone   = "done"
)

var Task = Category{
	Name: "task",
	Subcategories: SubcategoryList{
		{
			Name: taskNext,
			Transitions: []Transition{
				{Event: "notnow", Target: taskNotnow},
				{Event: "done", Target: taskDone},
			},
		},
		{
			Name: taskNotnow,
			Transitions: []Transition{
				{Event: "ready", Target: taskNext},
			},
		},
		{
			Name: taskDone,
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
	Name: "reference",
	Subcategories: SubcategoryList{
		{
			Name: referenceProcess,
		},
	},
}

var Idea = Category{
	Name: "idea",
	Subcategories: SubcategoryList{
		{
			Name: "default",
		},
	},
}

var People = Category{
	Name: "people",
	Subcategories: SubcategoryList{
		{
			Name: "default",
		},
	},
}

var Categories = CategoryList{Inbox, Reference, Idea, Task, People}
var RefileCategories = CategoryList{Inbox, Reference, Idea, Task}
