package email

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Mail struct {
	Body    string
	Subject string
	To      string
}

type Client interface {
	Send(context.Context, Mail) error
}

type CourierClient struct {
	token string
}

const courierBase = "https://api.courier.com"

func NewCourierClient(token string) *CourierClient {
	return &CourierClient{
		token,
	}
}

type To struct {
	Email string `json:"email"`
}

type Content struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Message struct {
	To      To      `json:"to"`
	Content Content `json:"content"`
}

type CourierMessage struct {
	Message `json:"message"`
}

func (cc *CourierClient) Send(ctx context.Context, mail Mail) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := CourierMessage{Message{
		To: To{
			Email: mail.To,
		},
		Content: Content{
			Title: mail.Subject,
			Body:  mail.Body,
		},
	}}

	r, w := io.Pipe()
	go func() {
		if err := json.NewEncoder(w).Encode(msg); err != nil {
			w.CloseWithError(err)
		}
		w.Close()
	}()

	defer r.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/send", courierBase), r)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cc.token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		response, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("request statuscode: %d read resposne: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("request statuscode %d: %s", resp.StatusCode, response)
	}

	return nil
}
