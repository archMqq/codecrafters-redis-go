package echo

func Exec(resp chan<- interface{}) {
	resp <- "+PONG\r\n"
}
