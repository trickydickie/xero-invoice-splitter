//go:generate go-winres make -in=winres.json
package main

import (
	"archive/zip"
	"fmt"
	"os"
	"path"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/pdfcpu/pdfcpu/pkg/api"
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

func splitInvoices(inFile string, outDir string, zip bool, dlg dialog.CustomDialog, prog *widget.ProgressBar) (err error) {
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		fmt.Println(err)
		dlg.Hide()
		return err
	}
	pageObj, _ := ctx.XRefTable.DereferenceDict(ctx.XRefTable.RootDict["Pages"])
	invObjects, _ := ctx.XRefTable.DereferenceArray(pageObj["Kids"])

	var pageSplits []int
	pageCount := 1
	dlg.Show()
	// len(invObjects)-1 since the last split is to the end of the file
	for i := 0; i < len(invObjects)-1; i++ {
		inv, _ := ctx.XRefTable.DereferenceDict(invObjects[i])
		pageSplits = append(pageSplits, (*inv.IntEntry("Count") + pageCount))
		pageCount += *inv.IntEntry("Count")
		prog.SetValue(float64(i+1)/float64(len(invObjects)-1))
	}
	err = api.SplitByPageNrFile(inFile, outDir, pageSplits, ctx.Configuration)

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

	return nil
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Xero Invoice Splitter")
	icon := canvas.NewImageFromResource(resourceIcon64Png)
	myWindow.SetIcon(icon.Resource)
	myWindow.Resize(fyne.NewSize(600, 400)) // Set default window size

	inputFilePath := ""
	outputFolderPath := ""

	// Text fields to display selected paths
	inputPathLabel := widget.NewLabel("No file selected")
	outputPathLabel := widget.NewLabel("No folder selected")

	// Button to open file picker dialog
	inputButton := widget.NewButton("Select Invoice File", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				inputFilePath = reader.URI().Path()
				inputPathLabel.SetText(inputFilePath)
			}
		}, myWindow)
	})

	// Button to open folder picker dialog
	outputButton := widget.NewButton("Select Output Folder", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				outputFolderPath = uri.Path()
				outputPathLabel.SetText(outputFolderPath)
			}
		}, myWindow)
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
		err := splitInvoices(inputFilePath, outputFolderPath, false, *dlg, progressBar)
		if err != nil{
			errDlg := dialog.NewError(err, myWindow)
			errDlg.Show()
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
