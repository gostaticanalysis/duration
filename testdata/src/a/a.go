package a

import "time"

func f() {
	var d1 time.Duration = 5 // want `must not use untyped constant as a time\.Duration type`
	time.Sleep(d1)

	d2 := 5 * time.Second // OK
	if true {
		d2 = 5 // want `must not use untyped constant as a time\.Duration type`
	}
	time.Sleep(d2)

	time.Sleep(5 * time.Second) // OK
}
