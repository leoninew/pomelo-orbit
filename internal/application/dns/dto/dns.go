package dnsdto

type Zone struct {
	Id   string
	Name string
}

type Record struct {
	Id        string
	Name      string
	Type      string
	Content   string
	TTL       int
	Proxied   bool
	Proxiable bool
}

type CreateRecordInput struct {
	Name string
	UpdateRecordInput
}

type UpdateRecordInput struct {
	Content string
	TTL     int
	Proxied bool
}
