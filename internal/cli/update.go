package cli

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	releasesURL = "https://github.com/wahyuakbarwibowo/martis/releases/latest"
	installURL  = "https://raw.githubusercontent.com/wahyuakbarwibowo/martis/main/install.sh"
)

// latestVersion reads the tag GitHub redirects /releases/latest to, e.g. "v0.6.1".
func latestVersion(url string) (string, error) {
	client := &http.Client{
		Timeout:       15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Head(url)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	_, tag, ok := strings.Cut(resp.Header.Get("Location"), "/tag/")
	if !ok || !strings.HasPrefix(tag, "v") {
		return "", fmt.Errorf("tidak bisa membaca versi rilis terbaru")
	}
	return tag, nil
}

// desktopInstalled reports whether the desktop app sits next to this install.
func desktopInstalled(binDir string) bool {
	for _, p := range []string{"/Applications/Martis.app", filepath.Join(os.Getenv("HOME"), "Applications", "Martis.app"), filepath.Join(binDir, "martis-desktop")} {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func selfUpdate(version string) {
	if runtime.GOOS == "windows" {
		fmt.Println("Update otomatis belum tersedia di Windows. Unduh versi terbaru dari:")
		fmt.Println("  " + releasesURL)
		return
	}
	if version == "dev" {
		fmt.Println("Build dev tidak bisa update otomatis. Pasang versi rilis dengan:")
		fmt.Println("  curl -fsSL " + installURL + " | bash")
		return
	}
	fmt.Println("⚡ Memeriksa versi terbaru...")
	latest, err := latestVersion(releasesURL)
	if err != nil {
		fmt.Printf("Gagal memeriksa versi: %v\n", err)
		os.Exit(1)
	}
	current := "v" + strings.TrimPrefix(version, "v")
	if current == latest {
		fmt.Printf("✅ Martis sudah versi terbaru (%s)\n", latest)
		return
	}
	fmt.Printf("Memperbarui %s → %s\n\n", current, latest)

	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		fmt.Printf("Gagal menemukan lokasi martis: %v\n", err)
		os.Exit(1)
	}
	binDir := filepath.Dir(exe)
	cmd := exec.Command("bash", "-c", "curl -fsSL "+installURL+" | bash")
	cmd.Env = append(os.Environ(), "INSTALL_DIR="+binDir, "VERSION="+latest)
	if desktopInstalled(binDir) {
		cmd.Env = append(cmd.Env, "MARTIS_DESKTOP=1")
	}
	// Attach the terminal so sudo can prompt for a password.
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("\nUpdate gagal: %v\n", err)
		os.Exit(1)
	}
}
