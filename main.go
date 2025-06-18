package main

import (
	"compress/bzip2" // Package for bzip2 decompression
	"encoding/xml"   // Package for XML encoding/decoding
	"fmt"            // Package for formatted I/O
	"io"             // Package for I/O primitives
	"net/http"       // Package for HTTP client functionality
	"os"             // Package for OS functions (file creation)
	"strings"        // Package for string manipulation
)

// Doc represents the <doc> element in the output XML
type Doc struct {
	XMLName  xml.Name `xml:"doc"`      // XML element name
	Title    string   `xml:"title"`    // Title of the page
	URL      string   `xml:"url"`      // URL of the wiki page
	Abstract string   `xml:"abstract"` // First paragraph of the page
}

func main() {
	// 1. Define the URL of the compressed Wikipedia dump
	url := "https://dumps.wikimedia.org/enwiki/latest/enwiki-latest-pages-articles-multistream.xml.bz2"

	// 2. Send an HTTP GET request to download the compressed data
	resp, err := http.Get(url)
	if err != nil {
		panic(fmt.Errorf("failed to download dump: %w", err))
	}
	defer resp.Body.Close() // Ensure the response body is closed

	// 3. Verify a successful HTTP response
	if resp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("bad status: %s", resp.Status))
	}

	// 4. Create a bzip2 reader to decompress on-the-fly
	bzReader := bzip2.NewReader(resp.Body)

	// 5. Create the output file for the abstracts XML
	out, err := os.Create("abstracts.xml")
	if err != nil {
		panic(fmt.Errorf("failed to create output file: %w", err))
	}
	defer out.Close() // Ensure the output file is closed

	// 6. Write the XML header and opening <documents> tag
	fmt.Fprintf(out, xml.Header)
	fmt.Fprintln(out, "<documents>")

	// 7. Initialize the XML decoder to read from the decompressed stream
	dec := xml.NewDecoder(bzReader)

	// 8. Loop through tokens until EOF
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			panic(fmt.Errorf("XML token error: %w", err))
		}

		// 9. Filter for start elements named <page>
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "page" {
			continue // Not a <page> start element
		}

		// 10. Decode the entire <page> element into a temporary struct
		var p struct {
			Title    string `xml:"title"` // Page title
			Revision struct {
				Text string `xml:"text"` // Page content
			} `xml:"revision"`
		}
		if err := dec.DecodeElement(&p, &start); err != nil {
			panic(fmt.Errorf("failed to decode page element: %w", err))
		}

		// 11. Split the page text at the first blank line to get the abstract
		parts := strings.SplitN(p.Revision.Text, "\n\n", 2)
		abstract := strings.TrimSpace(parts[0])
		if len(abstract) == 0 {
			continue // Skip pages with empty abstracts
		}

		// 12. Construct the URL for the wiki page from its title
		wikiTitle := strings.ReplaceAll(p.Title, " ", "_")
		pageURL := "https://en.wikipedia.org/wiki/" + wikiTitle

		// 13. Create a Doc instance and marshal it to indented XML
		doc := Doc{
			Title:    p.Title,
			URL:      pageURL,
			Abstract: abstract,
		}
		output, err := xml.MarshalIndent(doc, "  ", "    ")
		if err != nil {
			panic(fmt.Errorf("failed to marshal Doc: %w", err))
		}

		// 14. Write the marshaled <doc> element to the output file
		fmt.Fprintln(out, string(output))
	}

	// 15. Write the closing </documents> tag
	fmt.Fprintln(out, "</documents>")

	// 16. Notify the user that processing is done
	fmt.Println("Done! abstracts.xml is ready.")
}
