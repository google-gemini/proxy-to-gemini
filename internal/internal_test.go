// Copyright 2024 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google-gemini/proxy-to-gemini/internal"
)

func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		code     int
		msg      string
		arg      []interface{}
		wantBody string
		wantLog  string
	}{
		{
			name:     "Bad request without args",
			method:   http.MethodPost,
			code:     http.StatusBadRequest,
			msg:      "failed to read request body",
			wantBody: "failed to read request body\n",
		},
		{
			name:     "Internal server error with args",
			method:   http.MethodGet,
			code:     http.StatusInternalServerError,
			msg:      "failed to generate content: %v",
			arg:      []interface{}{fmt.Errorf("generic error")},
			wantBody: "failed to generate content: generic error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			req := httptest.NewRequest(tt.method, "/", nil)

			internal.ErrorHandler(recorder, req, tt.code, tt.msg, tt.arg...)

			if recorder.Code != tt.code {
				t.Errorf("got status %v, want %v", recorder.Code, tt.code)
			}

			if recorder.Body.String() != tt.wantBody {
				t.Errorf("got body %v, want %v", recorder.Body.String(), tt.wantBody)
			}
		})
	}
}
