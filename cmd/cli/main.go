package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/wadjakorntonsri/go-url-shortener/pkg/adapters/repository/sqlite"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/config"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/core/domain"
)

func main() {
	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
	importCmd := flag.NewFlagSet("import", flag.ExitOnError)
	importFile := importCmd.String("file", "", "JSON file to import")

	if len(os.Args) < 2 {
		fmt.Println("expected 'export' or 'import' subcommands")
		os.Exit(1)
	}

	cfg := config.Load()
	repo, err := sqlite.NewSQLiteRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}

	switch os.Args[1] {
	case "export":
		exportCmd.Parse(os.Args[2:])
		doExport(repo)
	case "import":
		importCmd.Parse(os.Args[2:])
		if *importFile == "" {
			importCmd.PrintDefaults()
			os.Exit(1)
		}
		doImport(repo, *importFile)
	default:
		fmt.Println("expected 'export' or 'import' subcommands")
		os.Exit(1)
	}
}

func doExport(repo *sqlite.SQLiteRepository) {
	links, err := repo.Dump(context.Background())
	// Note: repo.List in original implementation filtered deleted and had limit.
	// We need a raw dump method or use List with huge limit and raw SQL?
	// For implementing quickly, I'll assume we might want to extend repository or just query directly.
	// But let's stick to using what we have or extend it.
	// Since I cannot easily change the repo interface right now without touching other files,
	// I will just cast to concrete if needed (bad practice) or just use List.
	// Actually, the user wants "easy migrate", which implies DUMP.
	// I will add a method to SQLiteRepository in this file or rely on `List` just for active ones?
	// Better: I'll use a direct query here since I imported the sqlite package.
	// Ah, I set up the repo struct, but fields are private.
	// To do this properly, I should add `Dump()` to the repository interface or implementation.
	// For now, I'll use the public List method, but it filters deleted.
	// I'll stick to exporting ACTIVE links for now unless I go back and add Dump to interface.

	if err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(links); err != nil {
		log.Fatalf("Encode failed: %v", err)
	}
}

func doImport(repo *sqlite.SQLiteRepository, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	var links []domain.Link
	if err := json.NewDecoder(file).Decode(&links); err != nil {
		log.Fatalf("Decode failed: %v", err)
	}

	ctx := context.Background()
	count := 0
	for _, l := range links {
		// We try to insert. If ID exists, we might need UPSERT or just skip ID to let auto-increment handle it?
		// Usually for migration we want to keep IDs or ShortCodes.
		// Since ShortCode is unique, we should check existence.
		existing, _ := repo.GetByShortCode(ctx, l.ShortCode)
		if existing != nil {
			log.Printf("Skipping existing code: %s", l.ShortCode)
			continue
		}

		// Create basic
		err := repo.Create(ctx, &l)
		if err != nil {
			log.Printf("Failed to import %s: %v", l.ShortCode, err)
		} else {
			count++
		}
	}
	log.Printf("Imported %d links", count)
}
