package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Blacklist represents a set of blacklisted items.
type Blacklist map[string]bool

// HashPassword generates a bcrypt hash for the given password.
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// CheckHashedPassword compares a password with its hashed version.
func CheckHashedPassword(password string, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// NewBlacklistFromURL creates a new Blacklist from a specified URL containing blacklisted items.
func NewBlacklistFromURL() (Blacklist, error) {
	// URL containing blacklisted items
	url := "https://raw.githubusercontent.com/OWASP/passfault/master/wordlists/wordlists/500-worst-passwords.txt"

	// Fetch the blacklist from the URL
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Initialize a new Blacklist
	bl := make(Blacklist)

	// Use a scanner to read each line from the response body
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		// Trim leading and trailing whitespaces from each line
		item := strings.TrimSpace(scanner.Text())
		// Add the item to the blacklist
		bl.Add(item)
	}

	bl.Add("Siifra123@")

	log.Println("Bl: ", bl)

	return bl, nil
}

// Add adds an item to the blacklist.
func (bl Blacklist) Add(item string) {
	bl[item] = true
}

// IsBlacklisted checks if an item is in the blacklist.
func (bl Blacklist) IsBlacklisted(item string) bool {
	return bl[item]
}
