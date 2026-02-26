package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/pennsieve/pennsieve-go-llm/llm"
)

func main() {
	log.Println("LLM Summarizer Processor starting")

	inputDir := os.Getenv("INPUT_DIR")
	outputDir := os.Getenv("OUTPUT_DIR")

	if inputDir == "" || outputDir == "" {
		log.Fatal("INPUT_DIR and OUTPUT_DIR environment variables are required")
	}

	log.Printf("Input directory: %s", inputDir)
	log.Printf("Output directory: %s", outputDir)

	// Initialize the LLM governor client
	gov := llm.NewGovernor()
	if !gov.Available() {
		log.Fatal("LLM Governor not available: LLM_GOVERNOR_FUNCTION is not set")
	}

	ctx := context.Background()

	// Find JSON files in the input directory
	jsonFiles, err := filepath.Glob(filepath.Join(inputDir, "*.json"))
	if err != nil {
		log.Fatalf("Failed to list JSON files: %v", err)
	}
	if len(jsonFiles) == 0 {
		log.Fatal("No JSON files found in input directory")
	}

	log.Printf("Found %d JSON file(s)", len(jsonFiles))

	for _, jsonFile := range jsonFiles {
		log.Printf("Processing: %s", filepath.Base(jsonFile))

		// Read the JSON file
		data, err := os.ReadFile(jsonFile)
		if err != nil {
			log.Fatalf("Failed to read %s: %v", jsonFile, err)
		}

		// Parse JSON to verify it's valid and pretty-print for the prompt
		var parsed interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			log.Fatalf("Invalid JSON in %s: %v", jsonFile, err)
		}

		prettyJSON, err := json.MarshalIndent(parsed, "", "  ")
		if err != nil {
			log.Fatalf("Failed to format JSON: %v", err)
		}

		// Send to Bedrock via the governor
		log.Println("Sending to LLM for summarization...")

		prompt := fmt.Sprintf(`The following JSON represents a dataset. Please provide a comprehensive summary that includes:

1. **Overview**: What this dataset contains and its purpose
2. **Structure**: The key fields and their types
3. **Content Summary**: A description of the data values and any patterns
4. **Potential Uses**: What this dataset could be used for

JSON content:
%s`, string(prettyJSON))

		resp, err := gov.Invoke(ctx, &llm.InvokeRequest{
			Model:     llm.ModelHaiku45,
			System:    "You are a data analyst. Summarize datasets clearly and concisely. Use plain text paragraphs, not markdown.",
			MaxTokens: 2048,
			Messages: []llm.Message{
				llm.UserMessage(llm.TextBlock(prompt)),
			},
		})
		if err != nil {
			if ge, ok := llm.IsGovernorError(err); ok {
				switch {
				case ge.IsBudgetExceeded():
					log.Fatalf("LLM budget exceeded: %s", ge.Msg)
				case ge.IsModelNotAllowed():
					log.Fatalf("Model not allowed. Available models: %v", ge.AllowedModels)
				default:
					log.Fatalf("Governor error [%s]: %s", ge.Code, ge.Msg)
				}
			}
			log.Fatalf("Failed to invoke LLM: %v", err)
		}

		summary := resp.Text()
		log.Printf("Received summary (%d chars, cost: $%.4f)", len(summary), resp.Usage.EstimatedCostUsd)

		// Generate PDF
		baseName := strings.TrimSuffix(filepath.Base(jsonFile), filepath.Ext(jsonFile))
		pdfPath := filepath.Join(outputDir, baseName+"-summary.pdf")

		if err := generatePDF(pdfPath, baseName, summary); err != nil {
			log.Fatalf("Failed to generate PDF: %v", err)
		}

		log.Printf("Written: %s", pdfPath)
	}

	log.Println("LLM Summarizer Processor complete")
}

func generatePDF(path, title, body string) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	// Title
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 12, fmt.Sprintf("Dataset Summary: %s", title), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Metadata line
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(128, 128, 128)
	pdf.CellFormat(0, 5, fmt.Sprintf("Generated %s by Pennsieve LLM Summarizer", time.Now().UTC().Format("2006-01-02 15:04 UTC")), "", 1, "L", false, 0, "")
	pdf.Ln(6)

	// Separator line
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(6)

	// Body text
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(0, 0, 0)
	pdf.MultiCell(0, 6, body, "", "L", false)

	return pdf.OutputFileAndClose(path)
}
