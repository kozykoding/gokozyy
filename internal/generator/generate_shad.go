package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func setupShadcnManualV4(frontendDir string) error {
	fmt.Println("◦ Setting up shadcn/ui (manual, Tailwind v4)...")

	// 1) Ensure tsconfig.json and tsconfig.app.json have the alias shadcn expects
	if err := patchRootTsconfig(frontendDir); err != nil {
		return fmt.Errorf("patch root tsconfig: %w", err)
	}
	if err := patchAppTsconfig(frontendDir); err != nil {
		return fmt.Errorf("patch app tsconfig: %w", err)
	}

	// 2) Add shadcn-related deps
	cmd := exec.Command("bun", "add",
		"lucide-react",
		"class-variance-authority",
		"clsx",
		"tailwind-merge",
		"tailwindcss-animate",
	)
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bun add shadcn deps: %w", err)
	}

	// 3) components.json (points to tailwind.config.ts and src/index.css)
	componentsJSON := `{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "default",
  "rsc": false,
  "tsx": true,
  "tailwind": {
    "config": "tailwind.config.ts",
    "css": "src/index.css",
    "baseColor": "neutral"
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib/utils"
  }
}
`
	if err := os.WriteFile(
		filepath.Join(frontendDir, "components.json"),
		[]byte(componentsJSON),
		0o644,
	); err != nil {
		return fmt.Errorf("write components.json: %w", err)
	}

	// 4) src/components/ui/button.tsx
	uiDir := filepath.Join(frontendDir, "src", "components", "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		return fmt.Errorf("create src/components/ui: %w", err)
	}

	button := `import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none",
  {
    variants: {
      variant: {
        default: "bg-neutral-900 text-neutral-50 hover:bg-neutral-800",
        outline: "border border-neutral-200 hover:bg-neutral-100",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 px-3",
        lg: "h-10 px-8",
      }
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    );
  }
);
Button.displayName = "Button";

export { Button, buttonVariants };
`
	if err := os.WriteFile(
		filepath.Join(uiDir, "button.tsx"),
		[]byte(button),
		0o644,
	); err != nil {
		return fmt.Errorf("write button.tsx: %w", err)
	}

	// 5) src/lib/utils.ts for cn()
	libDir := filepath.Join(frontendDir, "src", "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		return fmt.Errorf("create src/lib: %w", err)
	}

	utils := `import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
`
	if err := os.WriteFile(
		filepath.Join(libDir, "utils.ts"),
		[]byte(utils),
		0o644,
	); err != nil {
		return fmt.Errorf("write src/lib/utils.ts: %w", err)
	}

	fmt.Println("◦ shadcn/ui (manual v4) installed: components.json, src/components/ui/button.tsx, src/lib/utils.ts")
	return nil
}
