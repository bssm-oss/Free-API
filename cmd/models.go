package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models per provider",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available free models by provider:")
		fmt.Println()
		fmt.Println("  gemini:")
		fmt.Println("    gemini-2.5-flash       (default, 10 RPM, 250 RPD)")
		fmt.Println("    gemini-2.5-flash-lite   (15 RPM, 1000 RPD)")
		fmt.Println("    gemini-2.0-flash        (5 RPM, 100 RPD)")
		fmt.Println("    gemini-2.5-pro          (5 RPM, 100 RPD)")
		fmt.Println("    gemini-1.5-flash        (15 RPM, 1500 RPD)")
		fmt.Println()
		fmt.Println("  groq:")
		fmt.Println("    llama-3.3-70b-versatile (default, 30 RPM, 1000 RPD)")
		fmt.Println("    llama-3.1-8b-instant    (30 RPM, 14400 RPD)")
		fmt.Println("    llama-4-scout-17b-16e-instruct (30 RPM, 1000 RPD)")
		fmt.Println("    qwen-qwq-32b           (30 RPM, 1000 RPD)")
		fmt.Println()
		fmt.Println("  cerebras:")
		fmt.Println("    llama-3.3-70b           (default, 30 RPM, 14400 RPD)")
		fmt.Println("    llama-3.1-8b            (30 RPM, 14400 RPD)")
		fmt.Println("    qwen-3-32b             (30 RPM, 14400 RPD)")
		fmt.Println()
		fmt.Println("  mistral:")
		fmt.Println("    mistral-small-latest    (default, 1 RPS, 500K TPM)")
		fmt.Println("    mistral-large-latest    (1 RPS)")
		fmt.Println("    codestral-latest        (30 RPM, codestral.mistral.ai)")
		fmt.Println()
		fmt.Println("  openrouter (free models, append :free):")
		fmt.Println("    deepseek/deepseek-r1:free      (default)")
		fmt.Println("    qwen/qwen3-coder-480b:free")
		fmt.Println("    meta-llama/llama-3.3-70b-instruct:free")
		fmt.Println("    google/gemma-3-27b-it:free")
		fmt.Println("    mistralai/mistral-small-3.1-24b-instruct:free")
		fmt.Println()
		fmt.Println("  cohere:")
		fmt.Println("    command-r-plus          (default, 20 RPM)")
		fmt.Println("    command-r               (20 RPM)")
		fmt.Println("    command-a-03-2025       (20 RPM)")
		fmt.Println()
		fmt.Println("  github:")
		fmt.Println("    gpt-4o                  (default, 10 RPM, 50 RPD)")
		fmt.Println("    gpt-4.1                 (10 RPM, 50 RPD)")
		fmt.Println("    o4-mini                 (10 RPM, 50 RPD)")
		fmt.Println()
		fmt.Println("  Usage: freeapi chat -m <model> \"message\"")
		fmt.Println("  Or:    freeapi config set <provider>.model <model>")
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
