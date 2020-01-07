package cli

var cliStates = [...]string{"🙈", "🙉", "🙊", "🐵"}

const (
	prefixSucceeded = "●" // ✔ ✓ 🆗 👌 ☑ ✅
	prefixSkipped   = "○" // ● • ‣ ◦ ⁃ ○ ◯ ⭕ 💮
	prefixFailed    = "✖" // ⨯ × ✗ x X ☓ ✘
)
