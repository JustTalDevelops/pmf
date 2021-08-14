package main

// Package main implements an example on converting a PMF world to a modern Minecraft Bedrock world.

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/justtaldevelops/pmf/pmf"
	"time"
)

func main() {
	start := time.Now()

	pm, err := pmf.DecodePMF("example")
	if err != nil {
		panic(err)
	}
	prov, err := mcdb.New("output")
	if err != nil {
		panic(err)
	}
	err = pm.Convert(prov)
	if err != nil {
		panic(err)
	}
	err = prov.Close()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Converted PMF world in %v!\n", time.Now().Sub(start))
}
