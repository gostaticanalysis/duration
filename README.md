# duration

[![godoc.org][godoc-badge]][godoc]

`duration` finds using untyped constant as time.Duration.

```go
duration := 5 // 5 * time.Second is correct
time.Sleep(duration)
```

```sh
$ go vet -vettool=`which duration` main.go
TODO
```

<!-- links -->
[godoc]: https://godoc.org/github.com/gostaticanalysis/duration
[godoc-badge]: https://img.shields.io/badge/godoc-reference-4F73B3.svg?style=flat-square&label=%20godoc.org

