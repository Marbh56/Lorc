package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"unicode"
)

type TokenType int

const (
	TokenError        TokenType = iota
	TokenLeftBrace              // {
	TokenRightBrace             // }
	TokenLeftBracket            // [
	TokenRightBracket           // ]
	TokenString
	TokenNumber
	TokenBoolean
	TokenNull
	TokenColon
	TokenComma
	TokenEOF
)

type Token struct {
	Type    TokenType
	Literal string
}

type Lexer struct {
	reader *bufio.Reader
	char   rune
	err    error
}

func NewLexer(input io.Reader) *Lexer {
	l := &Lexer{
		reader: bufio.NewReader(input),
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	char, _, err := l.reader.ReadRune()
	if err != nil {
		l.char = 0
		l.err = err
		return
	}
	l.char = char
}

func (l *Lexer) skipWhitespace() {
	for l.char == ' ' || l.char == '\t' || l.char == '\n' || l.char == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readString() (string, error) {
	var result []rune

	l.readChar() // Skip the opening quote

	for l.char != '"' && l.err == nil {
		if l.char == '\\' {
			l.readChar()
			switch l.char {
			case '"', '\\', '/':
				result = append(result, l.char)
			case 'b':
				result = append(result, '\b')
			case 'f':
				result = append(result, '\f')
			case 'n':
				result = append(result, '\n')
			case 'r':
				result = append(result, '\r')
			case 't':
				result = append(result, '\t')
			default:
				return "", fmt.Errorf("invalid escape sequence: \\%c", l.char)
			}
		} else if l.char < 32 {
			return "", fmt.Errorf("invalid control character in string: %d", l.char)
		} else {
			result = append(result, l.char)
		}
		l.readChar()
	}

	if l.err == io.EOF {
		return "", fmt.Errorf("unterminated string")
	}

	l.readChar() // Skip the closing quote
	return string(result), nil
}

func (l *Lexer) readNumber() string {
	var result []rune

	// Handle negative numbers
	if l.char == '-' {
		result = append(result, l.char)
		l.readChar()
	}

	// Read integer part
	for l.err == nil && unicode.IsDigit(l.char) {
		result = append(result, l.char)
		l.readChar()
	}

	// Handle decimal point
	if l.char == '.' {
		result = append(result, l.char)
		l.readChar()

		// Read fractional part
		for l.err == nil && unicode.IsDigit(l.char) {
			result = append(result, l.char)
			l.readChar()
		}
	}

	// Handle exponent notation
	if l.char == 'e' || l.char == 'E' {
		result = append(result, l.char)
		l.readChar()

		// Handle exponent sign
		if l.char == '+' || l.char == '-' {
			result = append(result, l.char)
			l.readChar()
		}

		// Read exponent digits
		for l.err == nil && unicode.IsDigit(l.char) {
			result = append(result, l.char)
			l.readChar()
		}
	}

	return string(result)
}

func (l *Lexer) readIdentifier() string {
	var result []rune
	for l.err == nil && (unicode.IsLetter(l.char) || l.char == '_') {
		result = append(result, l.char)
		l.readChar()
	}
	return string(result)
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	if l.err == io.EOF {
		return Token{Type: TokenEOF}
	}

	var tok Token

	switch {
	case l.char == '{':
		tok = Token{Type: TokenLeftBrace, Literal: string(l.char)}
		l.readChar()
	case l.char == '}':
		tok = Token{Type: TokenRightBrace, Literal: string(l.char)}
		l.readChar()
	case l.char == '[':
		tok = Token{Type: TokenLeftBracket, Literal: string(l.char)}
		l.readChar()
	case l.char == ']':
		tok = Token{Type: TokenRightBracket, Literal: string(l.char)}
		l.readChar()
	case l.char == ':':
		tok = Token{Type: TokenColon, Literal: string(l.char)}
		l.readChar()
	case l.char == ',':
		tok = Token{Type: TokenComma, Literal: string(l.char)}
		l.readChar()
	case l.char == '"':
		if str, err := l.readString(); err != nil {
			tok = Token{Type: TokenError, Literal: err.Error()}
		} else {
			tok = Token{Type: TokenString, Literal: str}
		}
	case unicode.IsDigit(l.char) || l.char == '-':
		number := l.readNumber()
		tok = Token{Type: TokenNumber, Literal: number}
	case unicode.IsLetter(l.char):
		identifier := l.readIdentifier()
		switch identifier {
		case "true", "false":
			tok = Token{Type: TokenBoolean, Literal: identifier}
		case "null":
			tok = Token{Type: TokenNull, Literal: identifier}
		default:
			tok = Token{Type: TokenError, Literal: "invalid identifier: " + identifier}
		}
	default:
		tok = Token{Type: TokenError, Literal: string(l.char)}
	}

	return tok
}

type Parser struct {
	lexer *Lexer
	token Token
}

func NewParser(lexer *Lexer) *Parser {
	p := &Parser{lexer: lexer}
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.token = p.lexer.NextToken()
}

// parseArray parses a JSON array: []
func (p *Parser) parseArray() error {
	// Move past the opening bracket
	p.nextToken()

	// Handle empty array case
	if p.token.Type == TokenRightBracket {
		p.nextToken()
		return nil
	}

	// Parse values until we hit the closing bracket
	for {
		if err := p.parseValue(); err != nil {
			return err
		}

		if p.token.Type == TokenRightBracket {
			p.nextToken()
			return nil
		}

		if p.token.Type != TokenComma {
			return fmt.Errorf("expected ',' or ']', got '%s'", p.token.Literal)
		}

		p.nextToken()
	}
}

// parseValue parses any JSON value
func (p *Parser) parseValue() error {
	switch p.token.Type {
	case TokenString, TokenNumber, TokenBoolean, TokenNull:
		p.nextToken()
		return nil
	case TokenLeftBrace:
		return p.ParseObject()
	case TokenLeftBracket:
		return p.parseArray()
	default:
		return fmt.Errorf("expected value, got %v", p.token.Literal)
	}
}

// ParseObject parses a JSON object: {}
func (p *Parser) ParseObject() error {
	// Expect opening brace
	if p.token.Type != TokenLeftBrace {
		return fmt.Errorf("expected '{', got '%s'", p.token.Literal)
	}
	p.nextToken()

	// Handle empty object case
	if p.token.Type == TokenRightBrace {
		p.nextToken()
		if p.token.Type == TokenEOF {
			return nil
		}
		return nil
	}

	// Parse key-value pairs
	for {
		// Parse key (must be string)
		if p.token.Type != TokenString {
			return fmt.Errorf("expected string key, got %v", p.token.Literal)
		}
		p.nextToken()

		// Expect colon
		if p.token.Type != TokenColon {
			return fmt.Errorf("expected ':', got '%s'", p.token.Literal)
		}
		p.nextToken()

		// Parse value
		if err := p.parseValue(); err != nil {
			return fmt.Errorf("invalid value: %v", err)
		}

		// After a key-value pair, expect either comma or closing brace
		if p.token.Type == TokenRightBrace {
			p.nextToken()
			if p.token.Type == TokenEOF {
				return nil
			}
			return nil
		}

		if p.token.Type != TokenComma {
			return fmt.Errorf("expected ',' or '}', got '%s'", p.token.Literal)
		}
		p.nextToken()
	}
}

func main() {
	var input io.Reader = os.Stdin

	if len(os.Args) > 1 {
		file, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		input = file
	}

	lexer := NewLexer(input)
	parser := NewParser(lexer)

	if err := parser.ParseObject(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Valid JSON")
	os.Exit(0)
}
