package actors

var terminateChan chan struct{}

func SetTerminateChan(term chan struct{}) {
	terminateChan = term
}

func GetTerminateChan() chan struct{} {
	return terminateChan
}
