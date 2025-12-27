package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func patchViteConfigForTailwind(frontendDir string) error {
	vitePath := filepath.Join(frontendDir, "vite.config.ts")
	data, err := os.ReadFile(vitePath)
	if err != nil {
		return err
	}

	content := string(data)

	if strings.Contains(content, "@tailwindcss/vite") {
		return nil
	}

	// 1) Ensure import
	lines := strings.Split(content, "\n")
	importInserted := false
	for i, line := range lines {
		if strings.HasPrefix(line, "import react") && !importInserted {
			lines[i] = line + "\nimport tailwindcss from \"@tailwindcss/vite\";"
			importInserted = true
			break
		}
	}

	// 2) Ensure plugins array contains tailwindcss()
	for i, line := range lines {
		if strings.Contains(line, "plugins: [react(") || strings.Contains(line, "plugins: [react(") {
			// turn into plugins: [react(), tailwindcss()],
			if !strings.Contains(line, "tailwindcss(") {
				line = strings.Replace(line, "react()", "react(), tailwindcss()", 1)
				lines[i] = line
			}
			break
		}
	}

	newContent := strings.Join(lines, "\n")
	return os.WriteFile(vitePath, []byte(newContent), 0o644)
}

func writeViteConfigWithTailwindV4(frontendDir string) error {
	viteContent := `import path from "path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
`
	return os.WriteFile(
		filepath.Join(frontendDir, "vite.config.ts"),
		[]byte(viteContent),
		0o644,
	)
}

func ensureRootTsconfigAlias(frontendDir string) error {
	tsconfigPath := filepath.Join(frontendDir, "tsconfig.json")

	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return fmt.Errorf("read tsconfig.json: %w", err)
	}

	var ts map[string]any
	if err := json.Unmarshal(data, &ts); err != nil {
		return fmt.Errorf("parse tsconfig.json: %w", err)
	}

	compiler, _ := ts["compilerOptions"].(map[string]any)
	if compiler == nil {
		compiler = map[string]any{}
	}

	if _, ok := compiler["baseUrl"]; !ok {
		compiler["baseUrl"] = "."
	}

	paths, _ := compiler["paths"].(map[string]any)
	if paths == nil {
		paths = map[string]any{}
	}
	if _, ok := paths["@/*"]; !ok {
		paths["@/*"] = []any{"./src/*"}
	}

	compiler["paths"] = paths
	ts["compilerOptions"] = compiler

	out, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tsconfig.json: %w", err)
	}

	if err := os.WriteFile(tsconfigPath, out, 0o644); err != nil {
		return fmt.Errorf("write tsconfig.json: %w", err)
	}

	return nil
}

func patchRootTsconfig(frontendDir string) error {
	tsconfigPath := filepath.Join(frontendDir, "tsconfig.json")

	content := `{
  "files": [],
  "references": [
    { "path": "./tsconfig.app.json" },
    { "path": "./tsconfig.node.json" }
  ],
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
`
	return os.WriteFile(tsconfigPath, []byte(content), 0o644)
}

func patchAppTsconfig(frontendDir string) error {
	appPath := filepath.Join(frontendDir, "tsconfig.app.json")

	content := `{
  "compilerOptions": {
    "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.app.tsbuildinfo",
    "target": "ES2022",
    "useDefineForClassFields": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "verbatimModuleSyntax": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedSideEffectImports": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["src"]
}
`
	return os.WriteFile(appPath, []byte(content), 0o644)
}
