package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"martis/internal/httpclient"
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
			fmt.Println("Penggunaan: martis import <postman.json|openapi.json|postman-env.json>")
			return true
		}
		if name, dotenv, ok, err := importer.PostmanEnvironment(os.Args[2]); ok || err != nil {
			if err != nil {
				fmt.Printf("Gagal mengimpor environment: %v\n", err)
				return true
			}
			if name == "" {
				name = strings.TrimSuffix(filepath.Base(os.Args[2]), filepath.Ext(os.Args[2]))
			}
			dir := filepath.Join(repository.ConfigDir(), "environments")
			target := filepath.Join(dir, filepath.Base(name)+".env")
			if _, err := os.Stat(target); err == nil {
				fmt.Printf("Environment %s sudah ada, tidak ditimpa\n", target)
				return true
			}
			if err := os.MkdirAll(dir, 0700); err != nil {
				fmt.Printf("Gagal membuat folder environment: %v\n", err)
				return true
			}
			if err := os.WriteFile(target, []byte(dotenv), 0600); err != nil {
				fmt.Printf("Gagal menyimpan environment: %v\n", err)
				return true
			}
			fmt.Printf("Berhasil mengimpor environment ke %s\n", target)
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
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		env := fs.String("env", "", "nama environment (mis. prod)")
		_ = fs.Parse(os.Args[2:])
		if fs.NArg() < 1 {
			fmt.Println("Penggunaan: martis run [--env <nama>] <folder>")
			os.Exit(2)
		}
		vars, err := loadEnv(filepath.Join(repository.ConfigDir(), "environments"), *env)
		if err != nil {
			fmt.Printf("Gagal memuat environment: %v\n", err)
			os.Exit(2)
		}
		col, err := repository.NewFileCollectionRepository().Load()
		if err != nil {
			fmt.Printf("Gagal memuat collections: %v\n", err)
			os.Exit(2)
		}
		failed, err := RunFolder(col, fs.Arg(0), vars, httpclient.NewClient("Martis-CLI/"+version).Do, os.Stdout)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		if failed > 0 {
			os.Exit(1)
		}
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
		fmt.Println("  martis <url>       Buka TUI dengan URL tersebut")
		fmt.Println("  martis curl ...    Buka TUI dengan request dari perintah cURL")
		fmt.Println("  martis collections Tampilkan daftar request di collection")
		fmt.Println("  martis import <file> Impor Postman v2, OpenAPI 3, atau environment Postman")
		fmt.Println("  martis run [--env <nama>] <folder>  Jalankan semua request di folder (exit 1 jika ada yang gagal)")
		fmt.Println("  martis version     Tampilkan versi aplikasi")
		fmt.Println("")
		fmt.Println("Aplikasi desktop: jalankan 'martis-desktop' (pasang dengan MARTIS_DESKTOP=1 pada install.sh)")
		fmt.Println("  martis update      Perbarui aplikasi dari upstream")
		fmt.Println("  martis help        Tampilkan bantuan ini")
		return true
	}

	return false
}
