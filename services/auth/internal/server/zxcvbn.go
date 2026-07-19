// Package zxcvbn implements a password strength estimation algorithm
// inspired by the zxcvbn library. It estimates crack time using pattern
// detection: dictionary words, keyboard sequences, repeats, l33t
// substitutions, and dates. Returns a score 0-4.
//
// This is a pure Go implementation with no external dependencies.
package server

import (
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Common dictionary words used for pattern matching.
var dictWords = map[string]bool{
	"password": true, "admin": true, "welcome": true, "letmein": true,
	"monkey": true, "dragon": true, "master": true, "login": true,
	"princess": true, "qwerty": true, "solo": true, "passw0rd": true,
	"starwars": true, "trustno1": true, "iloveyou": true, "batman": true,
	"access": true, "hello": true, "charlie": true, "donald": true,
	"shadow": true, "michael": true, "football": true, "sunshine": true,
	"ashley": true, "bailey": true, "superman": true,
	"qazwsx": true, "ninja": true, "mustang": true, "samsung": true,
	"root": true, "toor": true, "user": true, "guest": true,
}

// Common keyboard sequences.
var keyboardRows = []string{
	"qwertyuiop", "asdfghjkl", "zxcvbnm",
	"1234567890", "0987654321",
}

// StrengthResult holds the estimated password strength.
type StrengthResult struct {
	Score       int      `json:"score"`        // 0-4
	CrackTime   string   `json:"crack_time"`   // human-readable
	CrackSeconds float64  `json:"crack_seconds"`
	Guesses     float64  `json:"guesses"`      // estimated guess count
	Patterns    []string `json:"patterns"`     // detected patterns
	Suggestions []string `json:"suggestions"`
	Warning     string   `json:"warning"`
}

// EstimateStrength analyzes a password and returns a strength estimate.
// The scoring is:
//   0 = extremely weak (instant crack)
//   1 = weak (seconds to minutes)
//   2 = fair (hours to days)
//   3 = strong (months to years)
//   4 = very strong (centuries+)
func EstimateStrength(password string) StrengthResult {
	if len(password) == 0 {
		return StrengthResult{
			Score: 0, CrackTime: "instant", Patterns: []string{},
			Suggestions: []string{"Use a password"}, Warning: "Empty password",
		}
	}

	result := StrengthResult{
		Patterns:    []string{},
		Suggestions: []string{},
	}

	// Detect patterns — each reduces entropy.
	guessesLog10 := 0.0

	// 1. Dictionary word check.
	lower := strings.ToLower(password)
	if dictWords[lower] {
		result.Patterns = append(result.Patterns, "dictionary")
		guessesLog10 += 1.0 // very few guesses needed
		result.Warning = "This is a commonly used password"
	}

	// Check for dictionary words as substrings.
	for word := range dictWords {
		if len(word) >= 4 && strings.Contains(lower, word) {
			if !sliceContains(result.Patterns, "dictionary") {
				result.Patterns = append(result.Patterns, "dictionary")
			}
			guessesLog10 += 2.0
			result.Warning = "Contains a common dictionary word"
		}
	}

	// 2. Keyboard sequence check.
	for _, row := range keyboardRows {
		for i := 0; i+3 <= len(row); i++ {
			seq := row[i : i+3]
			if strings.Contains(lower, seq) {
				if !sliceContains(result.Patterns, "keyboard_sequence") {
					result.Patterns = append(result.Patterns, "keyboard_sequence")
				}
				guessesLog10 += 1.5
				result.Warning = "Contains a keyboard sequence"
			}
		}
	}

	// 3. Repeated character check.
	if hasRepeats(password) {
		result.Patterns = append(result.Patterns, "repeats")
		guessesLog10 += 1.0
		if result.Warning == "" {
			result.Warning = "Contains repeated characters"
		}
	}

	// 4. L33t substitution detection.
	if hasL33t(password) {
		result.Patterns = append(result.Patterns, "l33t")
		guessesLog10 += 0.5 // slight entropy increase for l33t
	}

	// 5. Date pattern detection.
	if reDate.MatchString(password) {
		result.Patterns = append(result.Patterns, "date")
		guessesLog10 += 2.0
		if result.Warning == "" {
			result.Warning = "Contains a date pattern"
		}
	}

	// 6. All digits or all letters.
	allDigits := true
	allLetters := true
	for _, c := range password {
		if !unicode.IsDigit(c) {
			allDigits = false
		}
		if !unicode.IsLetter(c) {
			allLetters = false
		}
	}
	if allDigits && len(password) <= 8 {
		result.Patterns = append(result.Patterns, "all_digits")
		guessesLog10 -= 1.0
		if result.Warning == "" {
			result.Warning = "All-digit passwords are easy to crack"
		}
	}
	if allLetters && len(password) <= 6 {
		if result.Warning == "" {
			result.Warning = "Short all-letter passwords are weak"
		}
	}

	// Calculate base entropy from length + character variety.
	charsetSize := calcCharsetSize(password)
	if charsetSize == 0 {
		charsetSize = 26
	}

	// Entropy = length * log2(charsetSize).
	// guesses ≈ 2^entropy, but reduce based on patterns found.
	entropy := float64(len(password)) * math.Log2(float64(charsetSize))
	if entropy < 0 {
		entropy = 0
	}

	// Penalty for detected patterns.
	patternPenalty := float64(len(result.Patterns)) * 3.0
	effectiveEntropy := entropy - patternPenalty
	if effectiveEntropy < 1 {
		effectiveEntropy = 1
	}

	// If patterns found, use the more conservative estimate.
	// Only dictionary and keyboard sequences significantly weaken a password.
	// Date/l33t/repeats are minor signals — a 20-char password with "2024" is still strong.
	hasSignificantPattern := false
	for _, p := range result.Patterns {
		if p == "dictionary" || p == "keyboard_sequence" || p == "all_digits" {
			hasSignificantPattern = true
			break
		}
	}
	if hasSignificantPattern && guessesLog10 > 0 {
		// Use pattern-based estimate if it's weaker.
		patternGuesses := math.Pow(10, guessesLog10+2)
		entropyGuesses := math.Pow(2, effectiveEntropy)
		if patternGuesses < entropyGuesses {
			result.Guesses = patternGuesses
		} else {
			result.Guesses = entropyGuesses
		}
	} else {
		result.Guesses = math.Pow(2, effectiveEntropy)
	}

	// Apply pattern-adjusted guesses as floor.
	if len(result.Patterns) > 0 {
		minGuesses := math.Pow(10, guessesLog10+1)
		if minGuesses > result.Guesses {
			result.Guesses = minGuesses
		}
	}

	// Calculate crack time: guesses / 1e10 guesses per second (offline fast attack).
	guessesPerSecond := 1e10
	result.CrackSeconds = result.Guesses / guessesPerSecond

	// Score based on crack time.
	result.Score = scoreFromCrackTime(result.CrackSeconds)
	result.CrackTime = humanizeDuration(result.CrackSeconds)

	// Generate suggestions.
	result.Suggestions = generateSuggestions(password, result.Score, result.Patterns)

	return result
}

var (
	reDate   = regexp.MustCompile(`(19|20)\d{2}`)     // 2024, 1999
)

// hasRepeats checks for 3+ consecutive repeated characters.
func hasRepeats(s string) bool {
	if len(s) < 3 {
		return false
	}
	count := 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			count++
			if count >= 3 {
				return true
			}
		} else {
			count = 1
		}
	}
	return false
}

// hasL33t checks for l33t speak substitutions: @→a, 3→e, $→s, 0→o, 1→i.
func hasL33t(s string) bool {
	l33tChars := "@3$01!|+7"
	for _, c := range s {
		if strings.ContainsRune(l33tChars, c) {
			return true
		}
	}
	return false
}

// calcCharsetSize determines the character set size used in the password.
func calcCharsetSize(password string) int {
	hasLower, hasUpper, hasDigit, hasSpecial := false, false, false, false
	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	size := 0
	if hasLower {
		size += 26
	}
	if hasUpper {
		size += 26
	}
	if hasDigit {
		size += 10
	}
	if hasSpecial {
		size += 32
	}
	return size
}

// scoreFromCrackTime converts crack time (seconds) to a 0-4 score.
func scoreFromCrackTime(seconds float64) int {
	switch {
	case seconds < 1:           // instant
		return 0
	case seconds < 600:         // < 10 minutes
		return 1
	case seconds < 86400:       // < 1 day
		return 2
	case seconds < 86400*365:   // < 1 year
		return 3
	default:                    // > 1 year
		return 4
	}
}

// humanizeDuration converts seconds to a human-readable string.
func humanizeDuration(seconds float64) string {
	if seconds < 1 {
		return "instant"
	}
	if seconds < 60 {
		return "less than a minute"
	}
	if seconds < 3600 {
		return "minutes"
	}
	if seconds < 86400 {
		return "hours"
	}
	if seconds < 86400*30 {
		return "days"
	}
	if seconds < 86400*365 {
		return "months"
	}
	if seconds < 86400*365*100 {
		return "years"
	}
	if seconds < 86400*365*1000 {
		return "centuries"
	}
	return "millennia"
}

// generateSuggestions produces actionable feedback based on the password.
func generateSuggestions(password string, score int, patterns []string) []string {
	var suggestions []string

	if len(password) < 12 {
		suggestions = append(suggestions, "Use at least 12 characters")
	}
	if score < 3 {
		suggestions = append(suggestions, "Add more variety: mix uppercase, lowercase, numbers, and symbols")
	}
	if sliceContains(patterns, "dictionary") {
		suggestions = append(suggestions, "Avoid common dictionary words")
	}
	if sliceContains(patterns, "keyboard_sequence") {
		suggestions = append(suggestions, "Avoid keyboard sequences (qwerty, asdf)")
	}
	if sliceContains(patterns, "repeats") {
		suggestions = append(suggestions, "Avoid repeated characters (aaa, 111)")
	}
	if sliceContains(patterns, "date") {
		suggestions = append(suggestions, "Avoid including dates")
	}
	if score >= 3 {
		suggestions = append(suggestions, "Strong password")
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Consider using a passphrase for even better security")
	}

	return suggestions
}

func sliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// Ensure time import is used (for future expansion).
var _ = time.Now