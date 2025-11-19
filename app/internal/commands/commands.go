package commands

type cmd interface {
	Exec(resp chan<- interface{})
}
