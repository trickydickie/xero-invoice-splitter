//go:generate go-winres make -in=winres.json
package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func addFiles(w *zip.Writer, basePath string) {
	files, err := os.ReadDir(basePath)
	if err != nil {
		fmt.Println(err)
	}

	for _, file := range files {
		dat, err := os.ReadFile(path.Join(basePath, file.Name()))
		if err != nil {
			fmt.Println(err)
		}

		f, err := w.Create(file.Name())
		if err != nil {
			fmt.Println(err)
		}
		_, err = f.Write(dat)
		if err != nil {
			fmt.Println(err)
		}
	}
}
func zipFiles(targetDir string) (err error){
	outFile, err := os.Create("output.zip")
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)

	addFiles(w, targetDir)

	err = w.Close()
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func splitInvoices(ctx *model.Context, inFile string, outDir string, zip bool, dlg dialog.CustomDialog, prog *widget.ProgressBar, label *widget.Label) (err error) {

	pageObj, _ := ctx.XRefTable.DereferenceDict(ctx.XRefTable.RootDict["Pages"])
	invObjects, _ := ctx.XRefTable.DereferenceArray(pageObj["Kids"])
	numInvoices := len(invObjects)

	var pageSplits []int
	pageCount := 1
	dlg.Show()
	// len(invObjects)-1 since the last split is to the end of the file
	for i := 0; i < numInvoices-1; i++ {
		inv, _ := ctx.XRefTable.DereferenceDict(invObjects[i])
		pageSplits = append(pageSplits, (*inv.IntEntry("Count") + pageCount))
		pageCount += *inv.IntEntry("Count")
		prog.SetValue((float64(i+1)/float64(numInvoices-1))*0.9)
	}
	if len(pageSplits) == 0 {
		err = errors.New("No splits detected.")
		fmt.Println(err)
		dlg.Hide()
		return err
	}
	err = api.SplitByPageNrFile(inFile, outDir, pageSplits, ctx.Configuration)
	prog.SetValue(1.0)
	if err != nil {
		fmt.Println(err)
		dlg.Hide()
		return err
	}

	if zip {
		err = zipFiles(outDir)
		if err != nil {
			dlg.Hide()
			return err
		}
	}

	label.SetText("Successfully split "+strconv.Itoa(numInvoices)+" invoices!")

	return nil
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Xero Invoice Splitter")
	icon := canvas.NewImageFromResource(resourceIcon64Png)
	myWindow.SetIcon(icon.Resource)
	myWindow.Resize(fyne.NewSize(600, 400)) // Set default window size
	myWindow.SetFixedSize(true)

	inputFilePath := ""
	outputFolderPath := ""

	// Text fields to display selected paths
	inputPathLabel := widget.NewLabel("No file selected")
	outputPathLabel := widget.NewLabel("No folder selected")
	

	destDlg := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err == nil && uri != nil {
			outputFolderPath = uri.Path()
			outputPathLabel.SetText(outputFolderPath)
		}
	}, myWindow)

	openDlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err == nil && reader != nil {
			inputFilePath = reader.URI().Path()
			inputPathLabel.SetText(inputFilePath)  // TODO: Truncate this label if the path is super long
			mfileURI := storage.NewFileURI(path.Dir(inputFilePath))
			mfileLister, _ := storage.ListerForURI(mfileURI)
			destDlg.SetLocation(mfileLister)
		}
	}, myWindow)

	openDlg.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))

	// Button to open file picker dialog
	inputButton := widget.NewButton("Select Invoice File", func() {
		openDlg.Show()
	})

	// Button to open folder picker dialog
	outputButton := widget.NewButton("Select Output Folder", func() {
		destDlg.Show()
	})

	// Button to trigger processing
	processButton := widget.NewButton("Split Invoices", func() {
		progressBar := widget.NewProgressBar()
		resultLabel := widget.NewLabel("")
		content := container.NewVBox(
			progressBar,
			resultLabel,
		)
		dlg := dialog.NewCustom("Processing Invoices", "OK", content, myWindow)
		ctx, err := api.ReadContextFile(inputFilePath)
		if err != nil{
			errDlg := dialog.NewError(err, myWindow)
			errDlg.Show()
			return
		}
		err = validateFile(ctx)
		if err != nil {
			fmt.Println(err)
			dialog.ShowConfirm(
				"Validation Failed",
				"This document does not appear to be a valid Xero export.\nDo you want to try split it anyway?",
			func(resp bool){
				if resp {
					err = splitInvoices(ctx, inputFilePath, outputFolderPath, false, *dlg, progressBar, resultLabel)
					if err != nil{
						errDlg := dialog.NewError(err, myWindow)
						errDlg.Show()
					}
				}
			}, myWindow)
		} else {
			err = splitInvoices(ctx, inputFilePath, outputFolderPath, false, *dlg, progressBar, resultLabel)
			if err != nil{
				errDlg := dialog.NewError(err, myWindow)
				errDlg.Show()
			}
		}
	})

	// Layout buttons and labels
	content := container.NewVBox(
		container.NewHBox(inputButton, inputPathLabel),
		container.NewHBox(outputButton, outputPathLabel),
		container.NewCenter(processButton),
	)
	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}
