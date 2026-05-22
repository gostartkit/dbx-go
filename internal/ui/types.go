package ui

import (
	"pkg.gostartkit.com/dbx/internal/ui/editor"
)

type Completion = editor.Completion
type Suggestion = editor.Suggestion
type CompletionRequest = editor.CompletionRequest
type CompletionResult = editor.CompletionResult
type CompletionEdit = editor.CompletionEdit
type Buffer = editor.Buffer
type Position = editor.Position

var NewSingleLineCompletionRequest = editor.NewSingleLineCompletionRequest
