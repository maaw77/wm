// Package pdf содержит генерацию PDF-отчета по результатам проверки ссылок.
package pdf

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/jung-kurt/gofpdf"
)

// GenerateLinksStatusReport генерирует PDF-отчет со сводной таблицей URL|Status.
//
// linksStatus: ключ = URL, значение = "available" / "not available".
// Возвращает байты PDF-файла или ошибку генерации.
func GenerateLinksStatusReport(linksStatus map[string]string) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(10, 10, 10)
	p.AddPage()

	p.SetFont("Arial", "B", 14)
	p.Cell(0, 10, "Links report")
	p.Ln(12)

	// Заголовок таблицы.
	p.SetFont("Arial", "B", 11)
	colURL := 140.0
	colStatus := 40.0
	rowH := 7.0

	p.CellFormat(colURL, rowH, "URL", "1", 0, "L", false, 0, "")
	p.CellFormat(colStatus, rowH, "Status", "1", 0, "L", false, 0, "")
	p.Ln(-1)

	urls := make([]string, 0, len(linksStatus))
	for u := range linksStatus {
		urls = append(urls, u)
	}
	sort.Strings(urls)

	p.SetFont("Arial", "", 10)
	for _, u := range urls {
		st := linksStatus[u]

		x := p.GetX()
		y := p.GetY()

		p.MultiCell(colURL, rowH, u, "1", "L", false)

		newY := p.GetY()
		hUsed := newY - y
		if hUsed < rowH {
			hUsed = rowH
		}

		p.SetXY(x+colURL, y)
		p.MultiCell(colStatus, rowH, st, "1", "L", false)

		p.SetXY(x, y+hUsed)

		// Простейшая защита от выхода за пределы страницы.
		if p.GetY() > 280 {
			p.AddPage()
		}
	}

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}
