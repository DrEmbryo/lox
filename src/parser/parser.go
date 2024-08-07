package parser

import (
	"fmt"
	"slices"

	"github.com/DrEmbryo/lox/src/grammar"
)

type Parser struct {
	Tokens  []grammar.Token
	current int
}

func (parser *Parser) consume() grammar.Token {
	token := parser.Tokens[parser.current]
	parser.current++
	return token
}

func (parser *Parser) lookahead() grammar.Token {
	token := parser.Tokens[parser.current]
	return token
}

func (parser *Parser) lookbehind() grammar.Token {
	token := parser.Tokens[parser.current-1]
	return token
}

func (parser *Parser) compareTypes(tokenType int) bool {
	return tokenType != grammar.EOF && parser.lookahead().TokenType == tokenType
}

func (parser *Parser) expect(tokenType int, message string) grammar.LoxError {
	if parser.compareTypes(tokenType) {
		parser.current++
		return nil
	}
	return ParserError{Token: parser.lookahead(), Message: message, Position: parser.current}
}

func (parser *Parser) matchToken(tokenTypes ...int) bool {
	for _, tokenType := range tokenTypes {
		if parser.compareTypes(tokenType) {
			parser.current++
			return true
		}
	}
	return false
}

func (parser Parser) Parse() ([]grammar.Statement, grammar.LoxError) {
	statements := make([]grammar.Statement, 0)

	if len(parser.Tokens) == 0 {
		return statements, ParserError{Position: 0, Message: "source contains 0 tokens"}
	}

	for parser.current <= len(parser.Tokens)-1 && parser.lookahead().TokenType != grammar.EOF {
		stmt, err := parser.declaration()
		if err != nil {
			parser.sync()
			return nil, err
		}
		statements = append(statements, stmt)
	}

	return statements, nil
}

func (parser *Parser) statement() (grammar.Statement, grammar.LoxError) {
	switch {
	case parser.matchToken(grammar.PRINT):
		return parser.PrintStatement()
	case parser.matchToken(grammar.LEFT_BRACE):
		return parser.blockStatement()
	case parser.matchToken(grammar.WHILE):
		return parser.whileStatement()
	case parser.matchToken(grammar.FOR):
		return parser.forStatement()
	case parser.matchToken(grammar.IF):
		return parser.conditionalStatement()
	default:
		return parser.expressionStatement()
	}
}

func (parser *Parser) conditionalStatement() (grammar.Statement, grammar.LoxError) {
	var condition grammar.Expression
	var thenBranch grammar.Statement
	var elseBranch grammar.Statement
	var err grammar.LoxError

	err = parser.expect(grammar.RIGHT_PAREN, "Expect '(' before condition inside 'if' statement")
	if err != nil {
		return nil, err
	}
	condition, err = parser.expression()
	if err != nil {
		return nil, err
	}

	err = parser.expect(grammar.LEFT_PAREN, "Expect ')' after condition inside 'if' statement")
	if err != nil {
		return nil, err
	}

	thenBranch, err = parser.statement()
	if err != nil {
		return nil, err
	}

	if parser.matchToken(grammar.ELSE) {
		elseBranch, err = parser.statement()
		if err != nil {
			return nil, err
		}
	}

	return grammar.ConditionalStatement{Condition: condition, ThenBranch: thenBranch, ElseBranch: elseBranch}, nil
}

func (parser *Parser) blockStatement() (grammar.Statement, grammar.LoxError) {
	statements := make([]grammar.Statement, 0)

	for !parser.compareTypes(grammar.RIGHT_BRACE) {
		stmt, err := parser.declaration()
		if err != nil {
			return nil, err
		}
		statements = append(statements, stmt)
	}

	return grammar.BlockScopeStatement{Statements: statements}, parser.expect(grammar.RIGHT_BRACE, "Expect '}' after value")
}

func (parser *Parser) PrintStatement() (grammar.Statement, grammar.LoxError) {
	value, err := parser.expression()
	if err != nil {
		return nil, err
	}

	return grammar.PrintStatement{Value: value}, parser.expect(grammar.SEMICOLON, "Expect ';' after value")
}

func (parser *Parser) whileStatement() (grammar.Statement, grammar.LoxError) {
	err := parser.expect(grammar.LEFT_PAREN, "Expect '(' after 'while' keyword")
	if err != nil {
		return nil, err
	}

	condition, err := parser.expression()
	if err != nil {
		return nil, err
	}

	err = parser.expect(grammar.RIGHT_PAREN, "Expect ')' after while loop condition")
	if err != nil {
		return nil, err
	}

	body, err := parser.statement()

	return grammar.WhileLoopStatement{Condition: condition, Body: body}, err
}

func (parser *Parser) forStatement() (grammar.Statement, grammar.LoxError) {
	parser.expect(grammar.LEFT_PAREN, "Expect '(' after 'for' keyword")

	var initializer grammar.Statement
	var err grammar.LoxError
	switch {
	case parser.matchToken(grammar.SEMICOLON):
		initializer = nil
	case parser.matchToken(grammar.VAR):
		initializer, err = parser.variableDeclaration()
	default:
		initializer, err = parser.expressionStatement()
	}
	if err != nil {
		return nil, err
	}

	var condition grammar.Expression
	if !parser.compareTypes(grammar.SEMICOLON) {
		condition, err = parser.expression()
	}
	if err != nil {
		return nil, err
	}
	parser.expect(grammar.SEMICOLON, "Expect ';' after for loop condition")

	var increment grammar.Expression
	if !parser.compareTypes(grammar.RIGHT_PAREN) {
		increment, err = parser.expression()
	}
	if err != nil {
		return nil, err
	}
	parser.expect(grammar.RIGHT_PAREN, "Expect ')' after for loop increment")

	body, err := parser.statement()
	if err != nil {
		return nil, err
	}

	if increment != nil {
		stmts := make([]grammar.Statement, 2)
		stmts = append(stmts, body, grammar.ExpressionStatement{Expression: increment})
		body = grammar.BlockScopeStatement{Statements: stmts}
	}

	if condition == nil {
		condition = grammar.LiteralExpression{Literal: true}
	}

	body = grammar.WhileLoopStatement{Condition: condition, Body: body}

	if initializer != nil {
		stmts := make([]grammar.Statement, 2)
		stmts = append(stmts, initializer, body)
		body = grammar.BlockScopeStatement{Statements: stmts}
	}

	return body, err
}

func (parser *Parser) expressionStatement() (grammar.Statement, grammar.LoxError) {
	expr, err := parser.expression()
	if err != nil {
		return nil, err
	}

	return grammar.ExpressionStatement{Expression: expr}, parser.expect(grammar.SEMICOLON, "Expect ';' after expression")
}

func (parser *Parser) expression() (grammar.Expression, grammar.LoxError) {
	return parser.assignment()
}

func (parser *Parser) assignment() (grammar.Expression, grammar.LoxError) {
	expr, err := parser.logicOr()
	if err != nil {
		return nil, err
	}
	//sould be assignmentExpression instead get expressionStatement
	fmt.Printf("%T \n", expr)

	if parser.matchToken(grammar.EQUAL) {
		fmt.Println("assignment")
		equal := parser.lookbehind()
		value, err := parser.assignment()
		if err != nil {
			return nil, err
		}

		switch exprType := expr.(type) {
		case grammar.VariableDeclaration:
			return grammar.AssignmentExpression{Name: exprType.Name, Value: value}, nil
		default:
			return nil, ParserError{Token: equal, Message: "Invalid assignment target.", Position: parser.current}
		}
	}
	return expr, nil
}

func (parser *Parser) logicOr() (grammar.Expression, grammar.LoxError) {
	leftExpr, err := parser.logicAnd()
	if err != nil {
		return nil, err
	}

	for parser.matchToken(grammar.OR) {
		operator := parser.lookbehind()
		rightExpr, err := parser.logicAnd()
		leftExpr = grammar.LogicExpression{Left: leftExpr, Right: rightExpr, Operator: operator}
		return leftExpr, err
	}

	return leftExpr, err
}

func (parser *Parser) logicAnd() (grammar.Expression, grammar.LoxError) {
	leftExpr, err := parser.equality()
	if err != nil {
		return nil, err
	}

	for parser.matchToken(grammar.AND) {
		operator := parser.lookbehind()
		rightExpr, err := parser.equality()
		leftExpr = grammar.LogicExpression{Left: leftExpr, Right: rightExpr, Operator: operator}
		return leftExpr, err
	}

	return leftExpr, err
}

func (parser *Parser) declaration() (grammar.Statement, grammar.LoxError) {
	if parser.matchToken(grammar.VAR) {
		return parser.variableDeclaration()
	}
	return parser.statement()
}

func (parser *Parser) variableDeclaration() (grammar.Statement, grammar.LoxError) {
	var initializer grammar.Expression

	err := parser.expect(grammar.IDENTIFIER, "Expect variable name.")
	if err != nil {
		return nil, err
	}
	name := parser.lookbehind()

	if parser.matchToken(grammar.EQUAL) {
		initializer, err = parser.expression()
		if err != nil {
			return nil, err
		}
	}

	parser.expect(grammar.SEMICOLON, "Expect ';' after variable declaration.")
	if err != nil {
		return nil, err
	}

	return grammar.VariableDeclarationStatement{Name: name, Initializer: initializer}, nil
}

func (parser *Parser) equality() (grammar.Expression, grammar.LoxError) {
	leftExpr, err := parser.comparison()

	for parser.matchToken(grammar.BANG, grammar.EQUAL_EQUAL) {
		operator := parser.lookbehind()
		rightExpr, err := parser.comparison()
		leftExpr = grammar.BinaryExpression{Left: leftExpr, Right: rightExpr, Operator: operator}
		return leftExpr, err
	}
	return leftExpr, err
}

func (parser *Parser) comparison() (grammar.Expression, grammar.LoxError) {
	leftExpr, err := parser.term()

	for parser.matchToken(grammar.GREATER, grammar.GREATER_EQUAL, grammar.LESS, grammar.LESS_EQUAL) {
		operator := parser.lookbehind()
		rightExpr, err := parser.term()
		leftExpr = grammar.BinaryExpression{Left: leftExpr, Right: rightExpr, Operator: operator}
		return leftExpr, err
	}

	return leftExpr, err
}

func (parser *Parser) term() (grammar.Expression, grammar.LoxError) {
	leftExpr, err := parser.factor()

	for parser.matchToken(grammar.MINUS, grammar.PLUS) {
		operator := parser.lookbehind()
		rightExpr, err := parser.factor()
		leftExpr = grammar.BinaryExpression{Left: leftExpr, Right: rightExpr, Operator: operator}
		return leftExpr, err
	}

	return leftExpr, err
}

func (parser *Parser) factor() (grammar.Expression, grammar.LoxError) {
	leftExpr, err := parser.unary()

	for parser.matchToken(grammar.SLASH, grammar.STAR) {
		operator := parser.lookbehind()
		rightExpr, err := parser.unary()
		leftExpr = grammar.BinaryExpression{Left: leftExpr, Right: rightExpr, Operator: operator}
		return leftExpr, err
	}

	return leftExpr, err
}

func (parser *Parser) unary() (grammar.Expression, grammar.LoxError) {
	if parser.matchToken(grammar.BANG, grammar.MINUS) {
		operator := parser.lookbehind()
		rightExpr, err := parser.unary()
		return grammar.UnaryExpression{Right: rightExpr, Operator: operator}, err
	}
	return parser.primary()
}

func (parser *Parser) primary() (grammar.Expression, grammar.LoxError) {
	switch {
	case parser.matchToken(grammar.FALSE):
		return grammar.LiteralExpression{Literal: false}, nil
	case parser.matchToken(grammar.TRUE):
		return grammar.LiteralExpression{Literal: true}, nil
	case parser.matchToken(grammar.NULL):
		return grammar.LiteralExpression{Literal: nil}, nil
	case parser.matchToken(grammar.NUMBER, grammar.STRING):
		return grammar.LiteralExpression{Literal: parser.lookbehind().Lexeme}, nil
	case parser.matchToken(grammar.IDENTIFIER):
		return grammar.VariableDeclaration{Name: parser.lookbehind()}, nil
	case parser.matchToken(grammar.LEFT_PAREN):
		expr, _ := parser.expression()
		return grammar.GroupingExpression{Expression: expr}, parser.expect(grammar.RIGHT_PAREN, "Expect ')' after expression.")
	}
	return nil, ParserError{Position: parser.current, Message: "Expect expression.", Token: parser.lookahead()}
}

func (parser *Parser) sync() {
	token := parser.Tokens[parser.current]
	for token.TokenType != grammar.EOF {
		token = parser.consume()
		if parser.lookbehind().TokenType == grammar.SEMICOLON || slices.Contains(grammar.SYNC_TOKENS, token.TokenType) {
			return
		}
	}
}
