package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const baseURL = "https://on.soundcloud.com/"

func main() {
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/random", handleRandomRequest)

	fmt.Println("Listening on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func handleRandomRequest(w http.ResponseWriter, r *http.Request) {
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // This stops the redirect
		},
	}
	for {
		randomPath := generateRandomString(5)
		attemptUrl := baseURL + randomPath

		fmt.Println("Trying ", attemptUrl)

		resp, err := client.Get(attemptUrl)
		if err != nil {
			fmt.Println("Error fetching URL:", err)
			continue
		}

		if resp.StatusCode == 302 { // HTTP 302 Found indicates a redirect
			resp.Body.Close()
			fullURL := resp.Header.Get("Location")

			if fullURL != "" {
				finalURL := strings.Split(fullURL, "?")[0] // Split at "?" and keep the first part
				fmt.Println("Found a hit:", finalURL)

				htmlContent := fetchHTML(finalURL)
				artistName, trackName, tag, description := extractInfo(htmlContent)

				if artistName == "\n" || trackName == "\n" {
					fmt.Println("Blank track data. Track probably removed")
					continue
				}
				
				// Save for later
				appendToJsonl("found.jsonl", finalURL, artistName, trackName, tag, description)
				
				embedString := generateEmbedString(finalURL)
				fmt.Fprintf(w, embedString)
				return
			}
		}
		resp.Body.Close()
	}
}

func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func fetchHTML(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching URL:", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return ""
	}
	return string(body)
}

func extractInfo(html string) (string, string, string, string) {
	// Scrape artist name
	artistNameStart := strings.Index(html, "by <a href=") + 10
	artistName := safeExtract(html, artistNameStart, "</a>")
	artistName = strings.Split(strings.Split(artistName, ">")[1], "<")[0]
	artistName = sanitizeAndUnescape(artistName)

	// Scrape track name
	trackNameStart := strings.Index(html, "<h1 itemprop=\"name\">") + 21
	trackName := safeExtract(html, trackNameStart, "</a>")
	trackName = strings.Split(strings.Split(trackName, ">")[1], "<")[0]
	trackName = sanitizeAndUnescape(trackName)

	// Scrape genre/tags
	tagStart := strings.Index(html, "<dd><a href=\"/tags/") + 19
	tags := safeExtract(html, tagStart, "</a>")
	tags = strings.Split(strings.Split(tags, ">")[1], "<")[0]
	tags = sanitizeAndUnescape(tags)

	// Scrape description
	descriptionStart := strings.Index(html, "<meta itemprop=\"description\" content=\"") + 38
	description := safeExtract(html, descriptionStart, "\" />")
	description = sanitizeAndUnescape(description)

	// Cut off description after some number of characters. 500 seems reasonable.
	if len(description) > 500 {
		description = description[:500]
	}

	return artistName, trackName, tags, description
}

func safeExtract(data string, start int, endMarker string) string {
	if start < 0 || start >= len(data) {
		return "" // Return blank if start index is out of range
	}
	endIndex := strings.Index(data[start:], endMarker)
	if endIndex == -1 {
		return "" // Return blank if end marker is not found
	}
	end := start + endIndex
	if end > len(data) {
		return "" // Ensure end does not exceed data length
	}
	return data[start:end]
}

func sanitizeAndUnescape(input string) string {
    // Replace Unicode escaped ampersand with an actual ampersand to correctly interpret HTML entities.
    // This step is crucial for cases like "\u0026#27;".
    input = strings.Replace(input, "\\u0026", "&", -1)

    // Now, unescape any HTML entities. This step will convert any HTML-encoded entities to their literal characters.
    unescapedHTML := html.UnescapeString(input)

    // Remove URLs, social media handles, emails, and implicit social media URLs.
    pattern := regexp.MustCompile(`https?://[^\s]+|@[a-zA-Z0-9_-]+|\b[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+|\bwww\.[^\s]+`)
    sanitized := pattern.ReplaceAllString(unescapedHTML, "")

    // Replace newlines with spaces to maintain readability without line breaks.
    sanitized = strings.Replace(sanitized, "\\n", " ", -1)

    // Check for and ignore erroneous HTML returns.
    if strings.HasPrefix(sanitized, "d>") {
        return ""
    }

    return sanitized
}

func generateEmbedString(targetURL string) string {
	// Embed format string with placeholder for the escaped URL
	embedFormat := `<iframe width="100%%" height="200" scrolling="no" frameborder="no" allow="autoplay" src="https://w.soundcloud.com/player/?url=%s&color=#282c34&auto_play=false&hide_related=false&show_comments=true&show_user=true&show_reposts=false&show_teaser=true&visual=true"></iframe>`

	return fmt.Sprintf(embedFormat, targetURL)
}


func appendToJsonl(fileName, url, artist, track, tags, description string) {
	// Create a single entry map to hold the data
	newEntry := map[string]string{
		"url":         url,
		"artist":      artist,
		"title":       track,
		"tags":        tags,
		"description": description,
	}

	// Marshal the map to a JSON object
	newEntryJson, err := json.Marshal(newEntry)
	if err != nil {
		fmt.Println("Error marshalling to JSON:", err)
		return
	}
	newEntryJson = []byte(strings.Replace(string(newEntryJson), "\\n", " ", -1))

	// Open the file in append mode, creating it if it doesn't exist
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Append the JSON object to the file, followed by a newline
	_, err = file.WriteString(string(newEntryJson) + "\n")
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
}

