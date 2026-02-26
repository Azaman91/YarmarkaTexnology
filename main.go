package main

import (
	"context"
	featureconnects "study/featureConnects"
)

func main() {
	ctx := context.Background()

	conn, err := featureconnects.Checkconnect(ctx)
	if err != nil {
		panic(err)
	}

	if err := featureconnects.Createtable(ctx, conn); err != nil {
		panic(err)
	}
}
