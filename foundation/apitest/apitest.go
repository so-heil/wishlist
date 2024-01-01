package apitest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const requestTimeout = 2 * time.Second

type EndpointTest struct {
	Name       string
	ReqBody    string
	StatusCode int
	RespDst    any
	Validate   func() error
	Headers    map[string]string
}

type Group struct {
	Name   string
	URL    string
	Method string
	Tests  []EndpointTest
}

func (g *Group) Run(t *testing.T) {
	for _, tt := range g.Tests {
		t.Run(fmt.Sprintf("%s:%s", g.Name, tt.Name), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, g.Method, g.URL, strings.NewReader(tt.ReqBody))
			if err != nil {
				t.Errorf("create request: %s", err)
				return
			}

			if tt.Headers != nil {
				for k, v := range tt.Headers {
					req.Header.Set(k, v)
				}
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("do request: %s", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.StatusCode {
				t.Errorf("staus code does not match: want %d got %d", tt.StatusCode, resp.StatusCode)
				return
			}

			if tt.RespDst != nil {
				if err := json.NewDecoder(resp.Body).Decode(tt.RespDst); err != nil {
					t.Errorf("decode response to dst: %s", err)
					return
				}
			} else {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("read response body: %s", err)
					return
				}
				if len(body) != 0 {
					fmt.Printf("response %s\n", string(body))
				}
			}

			if tt.Validate != nil {
				if err := tt.Validate(); err != nil {
					t.Errorf("validate: %s", err)
					return
				}
			}
		})
	}
}
