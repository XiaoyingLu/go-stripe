package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/phpdave11/gofpdf"
	"github.com/phpdave11/gofpdf/contrib/gofpdi"
)

type Order struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"-"`
	Items     []Product
}

type Product struct {
	Name     string `json:"name"`
	Amount   int    `json:"amount"`
	Quantity int    `json:"quantity"`
}

func (app *application) CreateAndSendInvoice(w http.ResponseWriter, r *http.Request) {
	// receive json
	var order Order

	err := app.readJSON(w, r, &order)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// order.ID = 100
	// order.Email = "me@here.com"
	// order.FirstName = "John"
	// order.LastName = "Smith"
	// order.Items = []Products{
	// 	{"Widget1", 1000, 1},
	// 	{"Widget2", 2000, 2},
	// 	{"Widget3", 3000, 3},
	// }
	// order.CreatedAt = time.Now()

	// generate a pdf invoice
	err = app.CreateInvoicePDF(order)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// create mail attachment
	attachments := []string{
		fmt.Sprintf("./invoices/%d.pdf", order.ID),
	}

	// send mail with attachment
	err = app.SendMail("info@widget.com", order.Email, "Your invoice", "invoice", attachments, nil)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// send response
	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Message = fmt.Sprintf("Invoice %d.pdf created and sent to %s", order.ID, order.Email)
	app.writeJSON(w, http.StatusOK, resp, nil)
}

func (app *application) CreateInvoicePDF(order Order) error {
	pdf := gofpdf.New("P", "mm", "Letter", "") // orientation, unit, size, font
	pdf.SetMargins(10, 13, 10)
	pdf.SetAutoPageBreak(true, 0)

	importer := gofpdi.NewImporter()

	t := importer.ImportPage(pdf, "./pdf-templates/invoice.pdf", 1, "/MediaBox")
	pdf.AddPage()

	importer.UseImportedTemplate(pdf, t, 0, 0, 215.9, 0)

	// write info
	pdf.SetY(50)
	pdf.SetX(10)
	pdf.SetFont("Times", "", 11)
	pdf.CellFormat(97, 8, fmt.Sprintf("Attention: %s %s", order.FirstName, order.LastName), "", 0, "L", false, 0, "")
	pdf.Ln(5)
	pdf.CellFormat(97, 8, order.Email, "", 0, "L", false, 0, "")
	pdf.Ln(5)
	pdf.CellFormat(97, 8, order.CreatedAt.Format("2006-01-02"), "", 0, "L", false, 0, "")

	y := 93.0
	for i, item := range order.Items {
		pdf.SetX(58)
		pdf.SetY(y + float64(i*8.0))
		pdf.CellFormat(155, 8, item.Name, "", 0, "L", false, 0, "")
		pdf.SetX(166)
		pdf.CellFormat(20, 8, fmt.Sprintf("%d", item.Quantity), "", 0, "C", false, 0, "")

		pdf.SetX(185)
		pdf.CellFormat(20, 8, fmt.Sprintf("$%.2f", float32(item.Amount)/100.0), "", 0, "R", false, 0, "")
	}
	invoicePath := fmt.Sprintf("./invoices/%d.pdf", order.ID)
	// save pdf
	err := pdf.OutputFileAndClose(invoicePath)
	if err != nil {
		return err
	}
	return nil
}
