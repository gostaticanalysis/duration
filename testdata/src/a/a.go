package a

import (
	"context"
	"time"
)

func f() {
	var d1 time.Duration = 5 // want `must not use untyped constant as a time\.Duration type`
	time.Sleep(d1)

	d2 := 5 * time.Second // OK
	if true {
		d2 = 5 // want `must not use untyped constant as a time\.Duration type`
	}
	time.Sleep(d2)

	time.Sleep(5 * time.Second) // OK

	const i = 6
	time.Sleep(i)                              // want `must not use untyped constant as a time\.Duration type`
	time.Sleep(7)                              // want `must not use untyped constant as a time\.Duration type`
	time.Sleep(time.Duration(3600))            // OK
	time.Sleep(time.Duration(5) * time.Second) // OK
	time.Sleep(60 * 60)                        // want `must not use untyped constant as a time\.Duration type`
	time.Sleep(i * 60)                         // want `must not use untyped constant as a time\.Duration type`

	const c = 2i * 2i
	time.Sleep(10 + c) // want `must not use untyped constant as a time\.Duration type`

	context.WithTimeout(context.Background(), i) // want `must not use untyped constant as a time\.Duration type`

	(T{}).sleep(i) // want `must not use untyped constant as a time\.Duration type`
}

type T struct{}

func (T) sleep(d time.Duration) {
	time.Sleep(d)
}
