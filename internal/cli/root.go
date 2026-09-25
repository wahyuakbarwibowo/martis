package cli

import (
	"fmt"
	"os"

	"martis/internal/importer"
	"martis/internal/repository"
)

// HandleCLIArgs memproses argumen command-line sebelum masuk ke UI TUI
func HandleCLIArgs(version string) bool {
	if len(os.Args) < 2 {
		return false
	}

	cmd := os.Args[1]
	switch cmd {
	case "import":
		if len(os.Args) < 3 {
			fmt.Println("Penggunaan: martis import <postman.json|openapi.json>")
			return true
		}
		incoming, err := importer.File(os.Args[2])
		if err != nil {
			fmt.Printf("Gagal mengimpor collection: %v\n", err)
			return true
		}
		repo := repository.NewFileCollectionRepository()
		current, err := repo.Load()
		if err != nil {
			fmt.Printf("Gagal memuat collections: %v\n", err)
			return true
		}
		current.Folders = append(current.Folders, incoming.Folders...)
		if err := repo.Save(current); err != nil {
			fmt.Printf("Gagal menyimpan collections: %v\n", err)
			return true
		}
		fmt.Printf("Berhasil mengimpor %d folder dari %s\n", len(incoming.Folders), os.Args[2])
		return true
	case "version", "-v", "--version":
		fmt.Printf("martis %s\n", version)
		return true

	case "update", "--update":
		fmt.Println("⚡ Memeriksa dan memperbarui Martis dari upstream...")
		if _, err := os.Stat(".git"); err == nil {
			fmt.Println("Repo git terdeteksi. Menjalankan make update-upstream...")
			fmt.Println("Menjalankan: make update-upstream")
		} else {
			fmt.Println("Untuk update langsung tanpa repositori lokal, gunakan:")
			fmt.Println("  go install github.com/wahyuakbarwibowo/martis@latest")
		}
		return true

	case "collections", "col":
		repo := repository.NewFileCollectionRepository()
		col, err := repo.Load()
		if err != nil {
			fmt.Printf("Gagal memuat collections: %v\n", err)
			return true
		}
		fmt.Printf("📁 Collections: %s (%d folders)\n\n", col.Name, len(col.Folders))
		for _, f := range col.Folders {
			fmt.Printf("📂 %s (%d requests)\n", f.Name, len(f.Items))
			for i, it := range f.Items {
				fmt.Printf("   %d. [%-6s] %-25s -> %s\n", i+1, it.Method, it.Name, it.URL)
			}
			fmt.Println()
		}
		return true

	case "help", "-h", "--help":
		fmt.Printf("Martis TUI - Ultra-Light REST Client (%s)\n\n", version)
		fmt.Println("Penggunaan:")
		fmt.Println("  martis             Buka Terminal User Interface (dengan Mouse Click Tree View)")
		fmt.Println("  martis collections Tampilkan daftar request di collection")
		fmt.Println("  martis import <file> Impor Postman v2 atau OpenAPI 3")
		fmt.Println("  martis version     Tampilkan versi aplikasi")
		fmt.Println("  martis update      Perbarui aplikasi dari upstream")
		fmt.Println("  martis help        Tampilkan bantuan ini")
		return true
	}

	return false
}
