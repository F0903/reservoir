package proxy

type Proxy interface {
	ListenBlocking(address string) error
	Listen(address string)
}
