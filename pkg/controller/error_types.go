package controller

import (
	"fmt"
)

type FileProcessingError struct {
	Msg      string
	FilePath string
	LineNum  int
	DocID    int // the number of YAML document where the error originates from (0-based)
}

func (e *FileProcessingError) Error() string {
	errorMsg := fmt.Sprintf("In file %s", e.FilePath)
	if e.LineNum > 0 {
		errorMsg += fmt.Sprintf(", line %d", e.LineNum)
	}
	if e.DocID >= 0 {
		errorMsg += fmt.Sprintf(", document %d", e.DocID)
	}
	errorMsg += fmt.Sprintf(": %s", e.Msg)
	return errorMsg
}

func (e *FileProcessingError) File() string {
	return e.FilePath
}

func (e *FileProcessingError) LineNo() int {
	return e.LineNum
}

func (e *FileProcessingError) DocumentID() int {
	return e.DocID
}

type FileProcessingErrorList []*FileProcessingError

func (e FileProcessingErrorList) Error() string {
	errMsg := ""
	for _, err := range e {
		errMsg += err.Error() + "\n"
	}
	return errMsg
}
