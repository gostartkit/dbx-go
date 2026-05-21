package commandlang

type Range struct {
	StartRune int
	EndRune   int
}

type Node interface {
	Range() Range
}

type Program struct {
	Commands  []*CommandNode
	Pipelines []*PipelineNode
	Errors    []ParseError
	Tokens    []Token
}

type CommandNode struct {
	Name        string
	Subcommands []string
	Args        []*ArgNode
	Flags       []*FlagNode
	Positionals []*ArgNode
	RangeValue  Range
}

type ArgNode struct {
	Value      string
	Quoted     bool
	RangeValue Range
}

type FlagNode struct {
	Name       string
	Value      *ArgNode
	HasValue   bool
	UsesEquals bool
	RangeValue Range
}

type PipelineNode struct {
	Commands   []*CommandNode
	RangeValue Range
}

type ParseError struct {
	Message    string
	RangeValue Range
}

type SyntaxContext struct {
	Program       *Program
	Command       *CommandNode
	Node          Node
	CursorRune    int
	CommandPath   []string
	ParentPath    []string
	InCommandName bool
	InSubcommand  bool
	InArg         bool
	InFlagName    bool
	InFlagValue   bool
	CurrentFlag   string
	ArgIndex      int
}

func (p *Program) Range() Range {
	if p == nil || len(p.Commands) == 0 {
		return Range{}
	}
	return Range{
		StartRune: p.Commands[0].RangeValue.StartRune,
		EndRune:   p.Commands[len(p.Commands)-1].RangeValue.EndRune,
	}
}

func (n *CommandNode) Range() Range {
	if n == nil {
		return Range{}
	}
	return n.RangeValue
}

func (n *ArgNode) Range() Range {
	if n == nil {
		return Range{}
	}
	return n.RangeValue
}

func (n *FlagNode) Range() Range {
	if n == nil {
		return Range{}
	}
	return n.RangeValue
}

func (n *PipelineNode) Range() Range {
	if n == nil {
		return Range{}
	}
	return n.RangeValue
}

func (e ParseError) Range() Range {
	return e.RangeValue
}

func ParseProgram(input string) *Program {
	return ParseTokens(Lex(input))
}

func ParseTokens(tokens []Token) *Program {
	parser := commandParser{
		tokens: append([]Token(nil), tokens...),
	}
	return parser.parseProgram()
}

type commandParser struct {
	tokens []Token
	index  int
}

func (p *commandParser) parseProgram() *Program {
	program := &Program{
		Commands: make([]*CommandNode, 0),
		Errors:   make([]ParseError, 0),
		Tokens:   append([]Token(nil), p.tokens...),
	}

	pipeline := make([]*CommandNode, 0, 2)
	for {
		command := p.parseCommand(program)
		if command != nil {
			program.Commands = append(program.Commands, command)
			pipeline = append(pipeline, command)
		}

		token := p.peek()
		switch token.Type {
		case TokenPipe:
			p.next()
			continue
		case TokenNewline:
			p.next()
			if len(pipeline) > 1 {
				program.Pipelines = append(program.Pipelines, newPipelineNode(pipeline))
			}
			pipeline = pipeline[:0]
		case TokenEOF:
			if len(pipeline) > 1 {
				program.Pipelines = append(program.Pipelines, newPipelineNode(pipeline))
			}
			return program
		default:
			p.next()
			program.Errors = append(program.Errors, ParseError{
				Message:    "unexpected token",
				RangeValue: tokenRange(token),
			})
		}
	}
}

func (p *commandParser) parseCommand(program *Program) *CommandNode {
	var command *CommandNode
	for {
		token := p.peek()
		switch token.Type {
		case TokenEOF, TokenPipe, TokenNewline:
			if command != nil {
				annotateCommandNode(command, nil)
			}
			return command
		case TokenFlag:
			if command == nil {
				command = &CommandNode{}
			}
			command.Flags = append(command.Flags, p.parseFlag(program))
			command.RangeValue = extendRange(command.RangeValue, tokenRange(token))
		case TokenWord, TokenString, TokenError:
			if command == nil {
				command = &CommandNode{}
			}
			arg := p.parseArg(program)
			command.Positionals = append(command.Positionals, arg)
			command.RangeValue = extendRange(command.RangeValue, arg.Range())
		case TokenEquals:
			p.next()
			program.Errors = append(program.Errors, ParseError{
				Message:    "unexpected equals",
				RangeValue: tokenRange(token),
			})
			if command == nil {
				command = &CommandNode{}
			}
			command.RangeValue = extendRange(command.RangeValue, tokenRange(token))
		case TokenBackslash:
			p.next()
			program.Errors = append(program.Errors, ParseError{
				Message:    "unexpected backslash",
				RangeValue: tokenRange(token),
			})
			if command == nil {
				command = &CommandNode{}
			}
			command.RangeValue = extendRange(command.RangeValue, tokenRange(token))
		default:
			p.next()
		}
	}
}

func (p *commandParser) parseFlag(program *Program) *FlagNode {
	flagToken := p.next()
	flagNode := &FlagNode{
		Name:       flagToken.Literal,
		RangeValue: tokenRange(flagToken),
	}

	if p.peek().Type == TokenEquals {
		flagNode.UsesEquals = true
		eq := p.next()
		flagNode.RangeValue = extendRange(flagNode.RangeValue, tokenRange(eq))
		if isValueToken(p.peek()) {
			arg := p.parseArg(program)
			flagNode.Value = arg
			flagNode.HasValue = true
			flagNode.RangeValue = extendRange(flagNode.RangeValue, arg.Range())
			return flagNode
		}
		program.Errors = append(program.Errors, ParseError{
			Message:    "flag value is required",
			RangeValue: flagNode.RangeValue,
		})
		return flagNode
	}

	if isValueToken(p.peek()) {
		arg := p.parseArg(program)
		flagNode.Value = arg
		flagNode.HasValue = true
		flagNode.RangeValue = extendRange(flagNode.RangeValue, arg.Range())
	}

	return flagNode
}

func (p *commandParser) parseArg(program *Program) *ArgNode {
	token := p.next()
	arg := &ArgNode{
		Value:      token.Literal,
		Quoted:     token.Type == TokenString,
		RangeValue: tokenRange(token),
	}
	if token.Type == TokenError {
		program.Errors = append(program.Errors, ParseError{
			Message:    "unterminated quoted string",
			RangeValue: tokenRange(token),
		})
	}
	return arg
}

func (p *commandParser) peek() Token {
	if p.index >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.index]
}

func (p *commandParser) next() Token {
	token := p.peek()
	if p.index < len(p.tokens) {
		p.index++
	}
	return token
}

func BuildSyntaxContext(program *Program, cursorRune int, knownPaths [][]string) SyntaxContext {
	ctx := SyntaxContext{
		Program:    program,
		CursorRune: cursorRune,
	}
	if program == nil {
		ctx.InCommandName = true
		return ctx
	}

	for _, command := range program.Commands {
		annotateCommandNode(command, knownPaths)
	}

	command := activeCommand(program.Commands, cursorRune)
	if command == nil {
		ctx.InCommandName = true
		return ctx
	}
	ctx.Command = command
	ctx.CommandPath = append([]string(nil), commandPath(command)...)

	token := cursorToken(program.Tokens, cursorRune)
	positionals := command.Positionals
	pathValues := positionalsToValues(positionals)
	pathLen := exactCommandPathLength(pathValues, knownPaths)
	if pathLen == 0 && len(positionals) > 0 {
		pathLen = 1
	}

	for _, flag := range command.Flags {
		if flag.UsesEquals && !flag.HasValue && token != nil && token.Type == TokenEquals && tokenInRange(token, flag.Range()) {
			ctx.Node = flag
			ctx.InFlagValue = true
			ctx.CurrentFlag = flag.Name
			ctx.ParentPath = append([]string(nil), commandPath(command)...)
			return ctx
		}
		if tokenInRange(token, flag.Range()) {
			ctx.Node = flag
			ctx.InFlagName = true
			ctx.CurrentFlag = flag.Name
			ctx.ParentPath = append([]string(nil), commandPath(command)...)
			return ctx
		}
		if flag.Value != nil && tokenInRange(token, flag.Value.Range()) {
			ctx.Node = flag.Value
			ctx.InFlagValue = true
			ctx.CurrentFlag = flag.Name
			ctx.ParentPath = append([]string(nil), commandPath(command)...)
			return ctx
		}
	}

	for index, arg := range positionals {
		if tokenInRange(token, arg.Range()) {
			ctx.Node = arg
			if index == 0 {
				ctx.InCommandName = true
				ctx.ParentPath = nil
				ctx.CommandPath = []string{arg.Value}
				return ctx
			}

			prefixBefore := pathValues[:index]
			if hasCommandChildren(prefixBefore, knownPaths) {
				ctx.InSubcommand = true
				ctx.ParentPath = append([]string(nil), prefixBefore...)
				ctx.CommandPath = append([]string(nil), prefixBefore...)
				return ctx
			}

			if index < pathLen {
				ctx.InSubcommand = true
				ctx.ParentPath = append([]string(nil), pathValues[:index]...)
				ctx.CommandPath = append([]string(nil), pathValues[:index+1]...)
				return ctx
			}

			ctx.InArg = true
			ctx.ArgIndex = index - pathLen
			ctx.ParentPath = append([]string(nil), commandPath(command)...)
			return ctx
		}
	}

	if previousFlag := pendingFlagValue(command.Flags, cursorRune); previousFlag != nil {
		ctx.Node = previousFlag
		ctx.InFlagValue = true
		ctx.CurrentFlag = previousFlag.Name
		ctx.ParentPath = append([]string(nil), commandPath(command)...)
		return ctx
	}

	beforeCount := positionalsBeforeCursor(positionals, cursorRune)
	if beforeCount == 0 {
		ctx.InCommandName = true
		return ctx
	}

	prefix := pathValues[:beforeCount]
	if hasCommandChildren(prefix, knownPaths) {
		ctx.InSubcommand = true
		ctx.ParentPath = append([]string(nil), prefix...)
		ctx.CommandPath = append([]string(nil), prefix...)
		return ctx
	}

	if beforeCount < pathLen {
		ctx.InSubcommand = true
		ctx.ParentPath = append([]string(nil), pathValues[:beforeCount]...)
		ctx.CommandPath = append([]string(nil), pathValues[:beforeCount]...)
		return ctx
	}

	ctx.InArg = true
	ctx.ArgIndex = beforeCount - pathLen
	ctx.ParentPath = append([]string(nil), commandPath(command)...)
	return ctx
}

func annotateCommandNode(command *CommandNode, knownPaths [][]string) {
	if command == nil || len(command.Positionals) == 0 {
		return
	}
	values := positionalsToValues(command.Positionals)
	pathLen := exactCommandPathLength(values, knownPaths)
	if pathLen == 0 {
		pathLen = 1
	}
	command.Name = values[0]
	command.Subcommands = append([]string(nil), values[1:pathLen]...)
	command.Args = append([]*ArgNode(nil), command.Positionals[pathLen:]...)
}

func activeCommand(commands []*CommandNode, cursorRune int) *CommandNode {
	var active *CommandNode
	for _, command := range commands {
		if command == nil {
			continue
		}
		if cursorRune >= command.Range().StartRune {
			active = command
		}
		if cursorRune >= command.Range().StartRune && cursorRune <= command.Range().EndRune {
			return command
		}
	}
	return active
}

func pendingFlagValue(flags []*FlagNode, cursorRune int) *FlagNode {
	for _, flag := range flags {
		if flag == nil || flag.HasValue {
			continue
		}
		if cursorRune >= flag.Range().EndRune {
			return flag
		}
	}
	return nil
}

func positionalsBeforeCursor(positionals []*ArgNode, cursorRune int) int {
	count := 0
	for _, arg := range positionals {
		if arg.Range().EndRune <= cursorRune {
			count++
		}
	}
	return count
}

func exactCommandPathLength(values []string, knownPaths [][]string) int {
	best := 0
	for _, candidate := range knownPaths {
		if len(candidate) == 0 || len(candidate) > len(values) {
			continue
		}
		if slicesEqual(candidate, values[:len(candidate)]) && len(candidate) > best {
			best = len(candidate)
		}
	}
	return best
}

func hasCommandChildren(prefix []string, knownPaths [][]string) bool {
	for _, candidate := range knownPaths {
		if len(prefix) == 0 {
			return len(candidate) > 0
		}
		if len(candidate) <= len(prefix) {
			continue
		}
		if slicesEqual(candidate[:len(prefix)], prefix) {
			return true
		}
	}
	return false
}

func positionalsToValues(positionals []*ArgNode) []string {
	values := make([]string, 0, len(positionals))
	for _, arg := range positionals {
		values = append(values, arg.Value)
	}
	return values
}

func commandPath(command *CommandNode) []string {
	if command == nil || command.Name == "" {
		return nil
	}
	path := []string{command.Name}
	path = append(path, command.Subcommands...)
	return path
}

func tokenRange(token Token) Range {
	return Range{StartRune: token.StartRune, EndRune: token.EndRune}
}

func extendRange(current Range, next Range) Range {
	if current.StartRune == 0 && current.EndRune == 0 {
		return next
	}
	if next.StartRune < current.StartRune {
		current.StartRune = next.StartRune
	}
	if next.EndRune > current.EndRune {
		current.EndRune = next.EndRune
	}
	return current
}

func tokenInRange(token *Token, current Range) bool {
	if token == nil {
		return false
	}
	return token.StartRune >= current.StartRune && token.EndRune <= current.EndRune
}

func slicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func newPipelineNode(commands []*CommandNode) *PipelineNode {
	if len(commands) == 0 {
		return nil
	}
	node := &PipelineNode{
		Commands: append([]*CommandNode(nil), commands...),
		RangeValue: Range{
			StartRune: commands[0].Range().StartRune,
			EndRune:   commands[len(commands)-1].Range().EndRune,
		},
	}
	return node
}
