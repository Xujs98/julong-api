package service

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const MaxDocumentTemplateUploadBytes = 10 * 1024 * 1024

// ParseModelDocumentUpload normalizes supported documents to Markdown. DOCX
// is read as bounded ZIP/XML data only: no macros, external resources, active
// HTML or embedded objects are executed or copied into the rendered document.
func ParseModelDocumentUpload(filename string, data []byte) (string, string, error) {
	if len(data) == 0 || len(data) > MaxDocumentTemplateUploadBytes {
		return "", "", common.NewMessage("Document file must be between 1 byte and 10 MB")
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".md" || ext == ".markdown" {
		content := strings.TrimPrefix(string(data), "\ufeff")
		if err := model.ValidateDocumentTemplateContent(content); err != nil {
			return "", "", err
		}
		return content, "markdown", nil
	}
	if ext != ".docx" {
		return "", "", common.NewMessage("Upload a Markdown (.md) or Word (.docx) document; save legacy .doc files as .docx first")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(archive.File) > 2048 {
		return "", "", common.NewMessage("Invalid Word document")
	}
	var document *zip.File
	var expanded uint64
	for _, file := range archive.File {
		if file.UncompressedSize64 > 16*1024*1024 || expanded > 16*1024*1024-file.UncompressedSize64 {
			return "", "", common.NewMessage("Word document expands beyond the allowed size")
		}
		expanded += file.UncompressedSize64
		if file.Name == "word/document.xml" {
			if document != nil || file.UncompressedSize64 > 8*1024*1024 {
				return "", "", common.NewMessage("Invalid Word document")
			}
			document = file
		}
	}
	if document == nil {
		return "", "", common.NewMessage("Invalid Word document")
	}
	reader, err := document.Open()
	if err != nil {
		return "", "", common.NewMessage("Invalid Word document")
	}
	defer reader.Close()
	data, err = io.ReadAll(io.LimitReader(reader, 8*1024*1024+1))
	if err != nil || len(data) > 8*1024*1024 {
		return "", "", common.NewMessage("Invalid Word document")
	}
	content, err := wordDocumentMarkdown(data)
	if err != nil {
		return "", "", err
	}
	if err := model.ValidateDocumentTemplateContent(content); err != nil {
		return "", "", err
	}
	return content, "docx", nil
}

func wordDocumentMarkdown(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var output, paragraph strings.Builder
	var table [][]string
	var row []string
	var cell []string
	depth, tableDepth, heading := 0, 0, 0
	inParagraph, inText, numbered, code := false, false, false, false
	const wordNamespace = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	for count := 0; ; count++ {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil || count > 1000000 {
			return "", common.NewMessage("Invalid Word document")
		}
		switch value := token.(type) {
		case xml.StartElement:
			depth++
			if depth > 128 {
				return "", common.NewMessage("Invalid Word document")
			}
			if value.Name.Space != wordNamespace {
				continue
			}
			switch value.Name.Local {
			case "tbl":
				tableDepth++
				if tableDepth > 1 {
					return "", common.NewMessage("Nested Word tables are not supported; upload Markdown instead")
				}
				table = nil
			case "tr":
				row = nil
			case "tc":
				cell = nil
			case "p":
				inParagraph, numbered, code, heading = true, false, false, 0
				paragraph.Reset()
			case "t":
				inText = inParagraph
			case "br", "cr":
				if inParagraph {
					paragraph.WriteByte('\n')
				}
			case "tab":
				if inParagraph {
					paragraph.WriteByte('\t')
				}
			case "numPr":
				numbered = true
			case "pStyle", "rFonts":
				for _, attr := range value.Attr {
					style := strings.ToLower(attr.Value)
					if value.Name.Local == "pStyle" && attr.Name.Local == "val" {
						if strings.HasPrefix(style, "heading") {
							heading, _ = strconv.Atoi(strings.TrimPrefix(style, "heading"))
							heading = min(max(heading, 0), 6)
						}
						code = code || strings.Contains(style, "code") || strings.Contains(style, "preformatted")
					}
					if value.Name.Local == "rFonts" {
						code = code || strings.Contains(style, "courier") || strings.Contains(style, "consolas") || strings.Contains(style, "mono")
					}
				}
			}
		case xml.CharData:
			if inText {
				paragraph.Write(value)
			}
		case xml.EndElement:
			depth--
			if value.Name.Space != wordNamespace {
				continue
			}
			switch value.Name.Local {
			case "t":
				inText = false
			case "p":
				inParagraph = false
				text := paragraph.String()
				if tableDepth > 0 {
					if len(cell) >= 2048 || len(text) > model.MaxDocumentTemplateContentBytes {
						return "", common.NewMessage("Word table exceeds the allowed size")
					}
					cell = append(cell, strings.TrimSpace(text))
				} else if text != "" {
					switch {
					case code:
						// Scan once; repeated substring searches become quadratic on
						// an untrusted paragraph consisting of long backtick runs.
						longest, run := 0, 0
						for _, char := range text {
							if char == '`' {
								run++
								longest = max(longest, run)
							} else {
								run = 0
							}
						}
						fenceLength := max(4, longest+1)
						if len(text)+2*fenceLength+4 > model.MaxDocumentTemplateContentBytes-output.Len() {
							return "", common.NewMessage("Document template content exceeds 2 MB")
						}
						fence := strings.Repeat("`", fenceLength)
						output.WriteString(fence + "\n" + text + "\n" + fence + "\n\n")
					case heading > 0:
						output.WriteString(strings.Repeat("#", heading) + " " + text + "\n\n")
					case numbered:
						output.WriteString("- " + text + "\n")
					default:
						output.WriteString(text + "\n\n")
					}
				}
			case "tc":
				if len(row) >= 2048 {
					return "", common.NewMessage("Word table exceeds the allowed size")
				}
				text := strings.Join(cell, "<br>")
				text = strings.ReplaceAll(strings.ReplaceAll(text, "|", "\\|"), "\n", "<br>")
				row = append(row, text)
			case "tr":
				if len(table) >= 4096 || len(row) >= 2048 {
					return "", common.NewMessage("Word table exceeds the allowed size")
				}
				table = append(table, row)
				if len(table) >= 4096 {
					return "", common.NewMessage("Word table exceeds the allowed size")
				}
			case "tbl":
				tableDepth--
				width := 0
				for _, cells := range table {
					width = max(width, len(cells))
				}
				// A ragged table can expand far beyond its small XML input
				// when padded to its widest row. Bound its exact output before
				// rectangularization or allocating the resulting Markdown.
				tableBytes := int64(1)
				for _, cells := range table {
					tableBytes += 5 + 3*int64(max(0, width-1))
					for _, text := range cells {
						tableBytes += int64(len(text))
					}
				}
				if len(table) > 0 {
					tableBytes += 2 + 6*int64(width)
				}
				if tableBytes > int64(model.MaxDocumentTemplateContentBytes-output.Len()) {
					return "", common.NewMessage("Document template content exceeds 2 MB")
				}
				for i, cells := range table {
					output.WriteString("| ")
					for column := range width {
						if column > 0 {
							output.WriteString(" | ")
						}
						if column < len(cells) {
							output.WriteString(cells[column])
						}
					}
					output.WriteString(" |\n")
					if i == 0 {
						output.WriteString("|" + strings.Repeat(" --- |", width) + "\n")
					}
				}
				output.WriteByte('\n')
			}
		}
		if output.Len() > model.MaxDocumentTemplateContentBytes {
			return "", common.NewMessage("Document template content exceeds 2 MB")
		}
	}
	return strings.TrimSpace(output.String()), nil
}
