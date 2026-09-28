package dun

import "log"

// logOperationError keeps recoverable errors visible in the terminal even
// when the UI also shows a dialog or continues with a safe fallback. Keep the
// operation name short and specific so console output is useful when several
// background/UI actions happen close together.
func logOperationError(operation string, err error) {
	if err != nil {
		log.Printf("dunnit: %s: %v", operation, err)
	}
}
