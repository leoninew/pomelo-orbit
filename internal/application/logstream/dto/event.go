package dto

type Event struct {
	Type     string
	SourceId string
	Cursor   string
	Data     []byte
	Status   string
	Message  string
}

type Emit func(Event) error
