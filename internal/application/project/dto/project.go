package dto

type CreateInput struct {
	Name string
	Code string
}

type SaveInput struct {
	Name string
}

// ProjectDefinition is the complete mutable business identity of a Project.
// Project and membership IDs are assigned and retained by the Project domain.
type ProjectDefinition struct {
	Name     string
	Code     string
	IsActive bool
}
