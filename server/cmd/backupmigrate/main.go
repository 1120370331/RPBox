package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/rpbox/server/internal/backup"
	"github.com/rpbox/server/internal/config"
)

func main() {
	apply := flag.Bool("apply", false, "upload and remove verified local dumps")
	limit := flag.Int("limit", 0, "maximum dumps to migrate; 0 means all")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	result, err := backup.MigrateLocalBackups(cfg, !*apply, *limit, func(message string) { fmt.Println(message) })
	if err != nil {
		log.Fatalf("migration stopped: %v; removed=%d uploaded=%d", err, result.Removed, result.Uploaded)
	}
	fmt.Printf("found=%d uploaded=%d removed=%d skipped=%d bytes_freed=%d\n",
		result.Found, result.Uploaded, result.Removed, result.Skipped, result.Bytes)
}
