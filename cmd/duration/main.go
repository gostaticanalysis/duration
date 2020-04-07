package main

import (
	"github.com/gostaticanalysis/duration"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() { unitchecker.Main(duration.Analyzer) }
