// This sample program demonstrates how to decode a JSON response
// using the json package and NewDecoder function.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type (
	// gResult maps to the result document received from the search.
	gResult struct {
		GsearchResultClass string `json:"GsearchResultClass"`
		UnescapedURL       string `json:"unescapedUrl"`
		URL                string `json:"url"`
		VisibleURL         string `json:"visibleUrl"`
		CacheURL           string `json:"cacheUrl"`
		Title              string `json:"title"`
		TitleNoFormatting  string `json:"titleNoFormatting"`
		Content            string `json:"content"`
	}

	// gResponse contains the top level document.
	gResponse struct {
		ResponseData struct {
			Results []gResult `json:"results"`
		} `json:"responseData"`
	}
)

func main() {
	uri := "https://api.duckduckgo.com/?q=golang&format=json"

	// Issue the search against Google.
	resp, err := http.Get(uri)
	if err != nil {
		log.Println("ERROR:", err)
		return
	}
	defer resp.Body.Close()

	// Decode the JSON response into our struct type.
	var gr gResponse
	err = json.NewDecoder(resp.Body).Decode(&gr)
	if err != nil {
		log.Println("ERROR:", err)
		return
	}

	fmt.Println(gr)

	// Marshal the struct type into a pretty print
	// version of the JSON document.
	pretty, err := json.MarshalIndent(gr, "", "    ")
	if err != nil {
		log.Println("ERROR:", err)
		return
	}

	fmt.Println(string(pretty))
}
