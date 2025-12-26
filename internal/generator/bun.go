// internal/generator/bun.go
package generator

import (
	"os"
	"os/exec"
)

// runBunCreateVite uses Bun to scaffold a Vite React app.
func runBunCreateVite(dir, name string) error {
	cmd := exec.Command("bunx", "create-vite@latest", name, "--template", "react-ts")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// bunInstall runs `bun install` in the given directory.
func bunInstall(dir string) error {
	cmd := exec.Command("bun", "install")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
