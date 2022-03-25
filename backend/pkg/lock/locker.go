package lock

type Locker interface {
	Lock(key string) (func(), error)
	RLock(key string) (func(), error)
}
