package dun

import (
	"context"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type llmCLIRequest struct {
	ctx      context.Context
	cancel   context.CancelFunc
	finished atomic.Bool
}

func newLLMCLIRequest() *llmCLIRequest {
	ctx, cancel := context.WithCancel(context.Background())
	return &llmCLIRequest{ctx: ctx, cancel: cancel}
}

func (r *llmCLIRequest) canceled() bool {
	return r.ctx.Err() == context.Canceled
}

func (r *llmCLIRequest) finish() {
	r.finished.Store(true)
}

func (r *llmCLIRequest) close() {
	if !r.finished.Load() {
		r.cancel()
	}
}

func (r *llmCLIRequest) stopButton() *widget.Button {
	var button *widget.Button
	button = widget.NewButton("Stop", func() {
		button.Disable()
		r.cancel()
	})
	return button
}

func llmCLIProgressContent(message string, request *llmCLIRequest) fyne.CanvasObject {
	return container.NewVBox(widget.NewLabel(message), request.stopButton())
}
