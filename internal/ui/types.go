package ui

import (
	"pkg.gostartkit.com/dbx/internal/commandlang"
	"pkg.gostartkit.com/dbx/internal/ui/editor"
)

type Completion = editor.Completion
type Suggestion = editor.Suggestion
type CompletionRequest = editor.CompletionRequest
type CompletionResult = editor.CompletionResult
type CompletionEdit = editor.CompletionEdit
type Completer = editor.Completer
type CommandContext = commandlang.CommandContext
type CommandToken = commandlang.Token
type Buffer = editor.Buffer
type Position = editor.Position

var NewSingleLineCompletionRequest = editor.NewSingleLineCompletionRequest
