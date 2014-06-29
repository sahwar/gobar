// gobar
// Copyright (C) 2014 Karol 'Kenji Takahashi' Woźniak
// 
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
// 
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
// 
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
// OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.


package main

import (
	"bufio"
	"io"
	"log"
	"regexp"
	"strconv"

	"github.com/BurntSushi/xgbutil/xgraphics"
)

type EndScan struct{}

func (e EndScan) Error() string { return "EndScan" }

func NewBGRA(color uint64) *xgraphics.BGRA {
	a := uint8(color >> 24)
	r := uint8((color & 0x00ff0000) >> 16)
	g := uint8((color & 0x0000ff00) >> 8)
	b := uint8(color & 0x000000ff)
	return &xgraphics.BGRA{B: b, G: g, R: r, A: a}
}

type TextPiece struct {
	Text       string
	Font       uint
	Foreground *xgraphics.BGRA
	Background *xgraphics.BGRA
	Screens    []uint
}

type TextParser struct {
	rgbPattern *regexp.Regexp
}

func NewTextParser() *TextParser {
	return &TextParser{regexp.MustCompile(`^0[xX][0-9a-fA-F]{8}$`)}
}

func (self *TextParser) Tokenize(
	data []byte, EOF bool,
) (advance int, token []byte, err error) {
	switch {
	case data[0] == '\n':
		err = EndScan{}
	case len(data) < 2:
		advance, token, err = 1, data[:1], nil
	case string(data[:2]) == "{F":
		advance, token, err = 2, data[:2], nil
	case string(data[:2]) == "{S":
		advance, token, err = 2, data[:2], nil
	case len(data) < 3:
		advance, token, err = 1, data[:1], nil
	case string(data[:3]) == "{CF":
		advance, token, err = 3, data[:3], nil
	case string(data[:3]) == "{CB":
		advance, token, err = 3, data[:3], nil
	case len(data) >= 10 && self.rgbPattern.Match(data[:10]):
		advance, token, err = 10, data[:10], nil
	case '0' <= data[0] && data[0] <= '9':
		i := 0
		for _, n := range data {
			if !('0' <= n && n <= '9') {
				break
			}
			i += 1
		}
		advance, token, err = i, data[:i], nil
	default: // Also contains '}' and ','
		// TODO: Parsing whole text piece here, instead of returning
		// char-by-char, should perform better
		advance, token, err = 1, data[:1], nil
	}
	return
}

// TODO: Escaping
func (self *TextParser) Scan(r io.Reader) ([]*TextPiece, error) {
	var text []*TextPiece

	scanner := bufio.NewScanner(r)

	scanner.Split(self.Tokenize)

	currentText := &TextPiece{}
	text = append(text, currentText)
	currentIndex := 0

	moveCurrent := func(delta int) *TextPiece {
		newCurrent := &TextPiece{}
		*newCurrent = *currentText
		newCurrent.Text = ""
		text = append(text, newCurrent)
		currentIndex += delta
		currentText = newCurrent
		return newCurrent
	}

	logPieceError := func(err error, pieces ...string) {
		log.Print(err)
		for _, piece := range pieces {
			currentText.Text += piece
		}
	}

	screening := false
	for scanner.Scan() {
		stext := scanner.Text()
		switch stext {
		case "{F":
			scanner.Scan()
			text := scanner.Text()
			font, err := strconv.Atoi(text)
			if err != nil {
				logPieceError(err, stext, text)
			}
			newCurrent := moveCurrent(1)
			newCurrent.Font = uint(font)
		case "{S":
			scanner.Scan()
			text := scanner.Text()
			screen, err := strconv.Atoi(text)
			if err != nil {
				logPieceError(err, stext, text)
			}
			newCurrent := moveCurrent(1)
			newCurrent.Screens = append(newCurrent.Screens, uint(screen))
			screening = true
		case "{CF":
			scanner.Scan()
			text := scanner.Text()
			fg, err := strconv.ParseUint(text, 0, 32)
			if err != nil {
				logPieceError(err, stext, text)
			}
			newCurrent := moveCurrent(1)
			newCurrent.Foreground = NewBGRA(fg)
		case "{CB":
			scanner.Scan()
			text := scanner.Text()
			bg, err := strconv.ParseUint(text, 0, 32)
			if err != nil {
				logPieceError(err, stext, text)
			}
			newCurrent := moveCurrent(1)
			newCurrent.Background = NewBGRA(bg)
		case "}":
			if len(text) > 1 {
				currentText = text[currentIndex - 1]
				moveCurrent(-1)
			}
			screening = false
		default:
			if screening && stext == "," {
				scanner.Scan()
				text := scanner.Text()
				screen, err := strconv.Atoi(text)
				if err != nil {
					logPieceError(err, stext, text)
				}
				currentText.Screens = append(currentText.Screens, uint(screen))
			} else {
				currentText.Text += stext
			}
		}
	}

	//Remove empty pieces. TODO: Merge with main parser
	var text2 []*TextPiece
	for _, piece := range text {
		if piece.Text != "" {
			text2 = append(text2, piece)
		}
	}

	return text2, nil
}
