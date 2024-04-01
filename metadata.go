package main

import (
	"errors"
	"fmt"
	"io"
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
