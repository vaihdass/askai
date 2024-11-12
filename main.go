package main

import (
	"fmt"
	md "github.com/MichaelMure/go-term-markdown"
	flag "github.com/spf13/pflag"
	"github.com/vaihdass/askai/internal/platform/duckduckgo"
	"github.com/yorukot/ansichroma"
	"log"
	"strings"
)

const (
	mdStyle      = "github"
	mdBackground = "#0d1117"
)

var (
	query    string
	model    string
	mdRender bool
	onlyCode bool
)

func init() {
	flag.StringVarP(&query, "query", "q", "Hello!", "Just a query")
	flag.StringVarP(&model, "model", "m", "claude-3-haiku-20240307",
		"Model to use: [gpt-4o-mini, mistralai/Mixtral-8x7B-Instruct-v0.1, meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo, claude-3-haiku-20240307]")
	flag.BoolVarP(&mdRender, "markdown", "md", false, "Render output as markdown")
	flag.BoolVarP(&onlyCode, "code", "c", false, "Try provide only code in non interactive mode")
}

func main() {
	flag.Parse()

	if onlyCode {
		query += "\nProvide only code! No additional comments!"
	}

	session := duckduckgo.NewSession(model)

	answerChan := getAIAnswer(session)
	if answerChan == nil {
		fmt.Println("Nothing...")
	}

	printAIAnswer(answerChan)
}

func getAIAnswer(s *duckduckgo.Session) <-chan string {
	if query == "" || len(query) == 0 {
		return nil
	}

	return s.Send(query)
}

func printAIAnswer(respChan <-chan string) {
	if onlyCode {
		printCode(respChan)
		return
	}

	if !mdRender {
		renderMarkdown(respChan)
		return
	}

	for c := range respChan {
		fmt.Print(c)
	}
}

func renderMarkdown(rawMsgChan <-chan string) {
	msg := strings.Builder{}

	for chunk := range rawMsgChan {
		msg.WriteString(chunk)
	}

	fmt.Println(string(md.Render(msg.String(), 80, 0)))
}

func printCode(answerChan <-chan string) {
	msg, lang := getCodeMsg(answerChan)

	if !mdRender {
		fmt.Println(msg)
		return
	}

	resultString, err := ansichroma.HightlightString(msg, lang, mdStyle, mdBackground)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resultString)
}

func getCodeMsg(codeChan <-chan string) (msg string, lang string) {
	var (
		msgBuilder strings.Builder
		line       strings.Builder
	)

	for chunk := range codeChan {
		line.WriteString(chunk)

		if !strings.Contains(chunk, "\n") {
			continue
		}

		strLine := line.String()

		if strings.Contains(strLine, "```") && len(strLine) > 3 {
			lang = strLine[3 : len(strLine)-1]
			line.Reset()
			continue
		}

		msgBuilder.WriteString(line.String())
		line.Reset()
	}

	return msgBuilder.String(), lang
}
