package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func setupTailwindV4(frontendDir string) error {
	fmt.Println("◦ Installing Tailwind CSS v4 (Vite plugin)...")

	// Install tailwindcss and the Vite plugin (and @types/node for TS tooling)
	cmd := exec.Command("bun", "add", "-D",
		"tailwindcss",
		"@tailwindcss/vite",
		"@types/node",
	)
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bun add tailwind v4 deps: %w", err)
	}

	// Tailwind v4-style config in TypeScript
	twConfig := `import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {},
  },
  plugins: [],
};

export default config;
`
	if err := os.WriteFile(
		filepath.Join(frontendDir, "tailwind.config.ts"),
		[]byte(twConfig),
		0o644,
	); err != nil {
		return fmt.Errorf("write tailwind.config.ts: %w", err)
	}

	// Tailwind v4 CSS entry: ensure @import "tailwindcss"; is at the top
	indexCSSPath := filepath.Join(frontendDir, "src", "index.css")
	existingIndex, err := os.ReadFile(indexCSSPath)
	if err != nil {
		// If the file doesn't exist for some reason, create a minimal one
		minimal := `@import "tailwindcss";
`
		if err := os.WriteFile(indexCSSPath, []byte(minimal), 0o644); err != nil {
			return fmt.Errorf("write src/index.css: %w", err)
		}
	} else {
		indexContent := string(existingIndex)
		if !strings.Contains(indexContent, `@import "tailwindcss";`) {
			indexContent = `@import "tailwindcss";
` + indexContent
			if err := os.WriteFile(indexCSSPath, []byte(indexContent), 0o644); err != nil {
				return fmt.Errorf("write src/index.css: %w", err)
			}
		}
	}

	// Ensure main.tsx imports index.css
	mainPath := filepath.Join(frontendDir, "src", "main.tsx")
	mainBytes, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("read main.tsx: %w", err)
	}
	if !strings.Contains(string(mainBytes), `./index.css`) {
		mainBytes = append(
			[]byte(`import "./index.css";`+"\n"),
			mainBytes...,
		)
		if err := os.WriteFile(mainPath, mainBytes, 0o644); err != nil {
			return fmt.Errorf("write main.tsx: %w", err)
		}
	}

	// Vite config with Tailwind plugin + @ alias
	if err := writeViteConfigWithTailwindV4(frontendDir); err != nil {
		return err
	}

	return nil
}
