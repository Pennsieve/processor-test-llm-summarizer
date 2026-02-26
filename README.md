# processor-test-llm-summarizer

A test Pennsieve processor that demonstrates LLM integration. It reads JSON dataset files, sends them to Amazon Bedrock for summarization via the [LLM Governor](https://github.com/pennsieve/pennsieve-go-llm), and outputs the results as PDF files.

## How it works

1. Reads all `*.json` files from the input directory
2. Sends each file to Claude Haiku 4.5 via the LLM Governor Lambda with a summarization prompt
3. Generates a formatted PDF for each file containing the summary
4. Writes the PDFs to the output directory as `{filename}-summary.pdf`

## Requirements

- The compute node must have LLM access enabled (`enableLLMAccess: true`)
- The `LLM_GOVERNOR_FUNCTION` and `EXECUTION_RUN_ID` environment variables are set automatically by the Pennsieve workflow orchestrator

## Environment variables

| Variable | Description | Set by |
|---|---|---|
| `INPUT_DIR` | Directory containing input JSON files | Workflow orchestrator |
| `OUTPUT_DIR` | Directory for output PDF files | Workflow orchestrator |
| `LLM_GOVERNOR_FUNCTION` | ARN of the LLM Governor Lambda | Workflow orchestrator |
| `EXECUTION_RUN_ID` | Current workflow execution ID | Workflow orchestrator |

## Build

```bash
docker build -t pennsieve/processor-test-llm-summarizer .
```

## Example input

Place a JSON file in the input directory:

```json
{
  "patients": [
    {"id": 1, "age": 45, "diagnosis": "hypertension", "visits": 12},
    {"id": 2, "age": 62, "diagnosis": "diabetes", "visits": 8}
  ],
  "study": "cardiovascular-risk-2024",
  "records": 2
}
```

The processor will produce a `patients-summary.pdf` containing a plain-text summary of the dataset's structure, content, and potential uses.

## Local testing

```bash
export INPUT_DIR=./testdata/input
export OUTPUT_DIR=./testdata/output
export LLM_GOVERNOR_FUNCTION=arn:aws:lambda:us-east-1:123456789:function:llm-governor
export EXECUTION_RUN_ID=test-run-001

mkdir -p $INPUT_DIR $OUTPUT_DIR
# Place JSON files in $INPUT_DIR

go run .
```