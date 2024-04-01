package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func validateFile(ctx *model.Context)(err error) {
	pageObj, _ := ctx.XRefTable.DereferenceDict(ctx.XRefTable.RootDict["Pages"])
	invObjects, _ := ctx.XRefTable.DereferenceArray(pageObj["Kids"])
	invCount := len(invObjects)
	valError := errors.New("File is not a valid Xero invoice export")

	mm, err := pdfcpu.ExtractMetadata(ctx)
	if err != nil{
		fmt.Println(err)
		return err
	}
	if len(mm) == 0 {
		return valError
	}
	
	meta, _ := io.ReadAll(mm[0].Reader)
	re := regexp.MustCompile(`<xero:oii>(.*?)</xero:oii>`)
	match := re.FindStringSubmatch(string(meta))
	if match != nil && len(match) > 1 {
		strings.Count(match[1],",")
		metaInvCount := strings.Count(match[1],",")+1
		if metaInvCount != invCount {
			return valError
		}
	} else {
		return valError
	}

	return nil
}

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

func truncatePath(filepath string, maxLen int) string {
	if len(filepath) <= maxLen {
		return filepath
	}

	truncatedPath := "..." + filepath[len(filepath)-maxLen+3:]

	return truncatedPath
}
