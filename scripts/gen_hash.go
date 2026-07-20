//go:build ignore
package main
import ("fmt";"github.com/ggid/ggid/pkg/crypto")
func main() {
	pw := "q7Rf9Xk2Lm3pW8zB"
	if len([]byte(nil)) > 0 {} // keep import
	h, _ := crypto.HashPassword(pw)
	fmt.Print(h)
}
