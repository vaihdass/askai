package duckduckgo

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

const (
	AIStatusURL = "https://duckduckgo.com/duckchat/v1/status"
	AIChatURL   = "https://duckduckgo.com/duckchat/v1/chat"
)

type Session struct {
	Context *Context
	VQD     string
	init    bool
	client  *http.Client
}

func NewSession(model string) *Session {
	s := Session{
		Context: NewContext(model),
		init:    false,
		client:  &http.Client{},
	}

	s.initSession()

	return &s
}
func (s *Session) initSession() {
	req, err := http.NewRequest("GET", AIStatusURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("x-vqd-accept", "1")
	resp, err := s.client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	s.VQD = resp.Header.Get("x-vqd-4")
	s.init = true
}
func (s *Session) Send(msg string) <-chan string {
	if !s.init {
		log.Fatal("session not initialized")
	}

	s.Context.Messages = append(s.Context.Messages, NewMessage("user", msg))
	contextJSON, err := json.Marshal(s.Context)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", AIChatURL, bytes.NewBuffer(contextJSON))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("x-vqd-4", s.VQD)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	s.VQD = resp.Header.Get("x-vqd-4")

	result := make(chan string)
	go s.runGetMessage(result, resp)

	return result
}

func (s *Session) runGetMessage(result chan<- string, resp *http.Response) {
	scanner := bufio.NewScanner(resp.Body)
	defer func() {
		_ = resp.Body.Close()
	}()

	msg := strings.Builder{}
	for scanner.Scan() {
		line := scanner.Text()

		splitLine := strings.Split(line, "data: ")
		if len(splitLine) > 1 {
			var answerContent map[string]any

			err := json.Unmarshal([]byte(splitLine[1]), &answerContent)
			if err != nil {
				continue
			}

			if chunk, ok := answerContent["message"]; ok {
				result <- chunk.(string)
				msg.WriteString(chunk.(string))
			}
		}
	}

	s.Context.Messages = append(s.Context.Messages, NewMessage("assistant", msg.String()))
	close(result)
}
