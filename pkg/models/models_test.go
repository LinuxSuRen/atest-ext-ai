package models

import (
	"encoding/json"
	"testing"
)

func TestAIRequest_JSON(t *testing.T) {
	tests := []struct {
		name    string
		request *AIRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &AIRequest{
				Action:    "convert_to_sql",
				Data:      map[string]interface{}{"query": "SELECT * FROM users", "provider": "openai"},
				RequestID: "test-123",
			},
			wantErr: false,
		},
		{
			name: "empty request",
			request: &AIRequest{
				Action:    "",
				Data:      map[string]interface{}{},
				RequestID: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Test unmarshaling
			var unmarshaled AIRequest
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("json.Unmarshal() error = %v", err)
				return
			}

			// Verify fields
			if unmarshaled.Action != tt.request.Action {
				t.Errorf("Action = %v, want %v", unmarshaled.Action, tt.request.Action)
			}
			if unmarshaled.RequestID != tt.request.RequestID {
				t.Errorf("RequestID = %v, want %v", unmarshaled.RequestID, tt.request.RequestID)
			}
			// Note: Data field comparison would require deep comparison
		})
	}
}

func TestAIResponse_JSON(t *testing.T) {
	tests := []struct {
		name     string
		response *AIResponse
		wantErr  bool
	}{
		{
			name: "valid response",
			response: &AIResponse{
				Success:   true,
				Data:      map[string]interface{}{"sql": "SELECT * FROM users WHERE id = 1"},
				Message:   "Query converted successfully",
				RequestID: "test-123",
			},
			wantErr: false,
		},
		{
			name: "error response",
			response: &AIResponse{
				Success:   false,
				Data:      map[string]interface{}{},
				Message:   "Failed to convert query",
				RequestID: "test-123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Test unmarshaling
			var unmarshaled AIResponse
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("json.Unmarshal() error = %v", err)
				return
			}

			// Verify fields
			if unmarshaled.Success != tt.response.Success {
				t.Errorf("Success = %v, want %v", unmarshaled.Success, tt.response.Success)
			}
			if unmarshaled.Message != tt.response.Message {
				t.Errorf("Message = %v, want %v", unmarshaled.Message, tt.response.Message)
			}
			if unmarshaled.RequestID != tt.response.RequestID {
				t.Errorf("RequestID = %v, want %v", unmarshaled.RequestID, tt.response.RequestID)
			}
			// Note: Data and Timestamp field comparison would require deep comparison
		})
	}
}

func TestSQLConversionRequest_JSON(t *testing.T) {
	request := &SQLConversionRequest{
		Query:       "show me all users",
		Context:     "user database",
		TableSchema: map[string]string{"users": "id, name, email"},
		Dialect:     "mysql",
	}

	// Test marshaling
	data, err := json.Marshal(request)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
		return
	}

	// Test unmarshaling
	var unmarshaled SQLConversionRequest
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	// Verify fields
	if unmarshaled.Query != request.Query {
		t.Errorf("Query = %v, want %v", unmarshaled.Query, request.Query)
	}
	if unmarshaled.Context != request.Context {
		t.Errorf("Context = %v, want %v", unmarshaled.Context, request.Context)
	}
	if unmarshaled.Dialect != request.Dialect {
		t.Errorf("Dialect = %v, want %v", unmarshaled.Dialect, request.Dialect)
	}
}

func TestSQLConversionResponse_JSON(t *testing.T) {
	response := &SQLConversionResponse{
		SQL:         "SELECT * FROM users",
		Confidence:  0.95,
		Explanation: "Conversion successful",
		Warnings:    []string{"Consider adding LIMIT clause"},
	}

	// Test marshaling
	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
		return
	}

	// Test unmarshaling
	var unmarshaled SQLConversionResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	// Verify fields
	if unmarshaled.SQL != response.SQL {
		t.Errorf("SQL = %v, want %v", unmarshaled.SQL, response.SQL)
	}
	if unmarshaled.Confidence != response.Confidence {
		t.Errorf("Confidence = %v, want %v", unmarshaled.Confidence, response.Confidence)
	}
	if unmarshaled.Explanation != response.Explanation {
		t.Errorf("Explanation = %v, want %v", unmarshaled.Explanation, response.Explanation)
	}
}

func TestHealthCheckResponse_JSON(t *testing.T) {
	response := &HealthCheckResponse{
		Status:   "healthy",
		Services: map[string]string{"ai": "healthy", "database": "healthy"},
		Version:  "1.0.0",
	}

	// Test marshaling
	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
		return
	}

	// Test unmarshaling
	var unmarshaled HealthCheckResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	// Verify fields
	if unmarshaled.Status != response.Status {
		t.Errorf("Status = %v, want %v", unmarshaled.Status, response.Status)
	}
	if unmarshaled.Version != response.Version {
		t.Errorf("Version = %v, want %v", unmarshaled.Version, response.Version)
	}
}

func TestModelInfo_JSON(t *testing.T) {
	modelInfo := &ModelInfo{
		Name:         "gpt-3.5-turbo",
		Provider:     "openai",
		Version:      "1.0",
		Capabilities: []string{"text-generation", "sql-conversion"},
		Limits:       map[string]int{"max_tokens": 4096},
		Metadata:     map[string]string{"type": "language_model"},
	}

	// Test marshaling
	data, err := json.Marshal(modelInfo)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
		return
	}

	// Test unmarshaling
	var unmarshaled ModelInfo
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	// Verify fields
	if unmarshaled.Name != modelInfo.Name {
		t.Errorf("Name = %v, want %v", unmarshaled.Name, modelInfo.Name)
	}
	if unmarshaled.Provider != modelInfo.Provider {
		t.Errorf("Provider = %v, want %v", unmarshaled.Provider, modelInfo.Provider)
	}
	if unmarshaled.Version != modelInfo.Version {
		t.Errorf("Version = %v, want %v", unmarshaled.Version, modelInfo.Version)
	}
}

func TestErrorResponse_JSON(t *testing.T) {
	errorResp := &ErrorResponse{
		Code:      "invalid_request",
		Message:   "The request is invalid",
		Details:   "Missing required field",
		RequestID: "test-123",
	}

	// Test marshaling
	data, err := json.Marshal(errorResp)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
		return
	}

	// Test unmarshaling
	var unmarshaled ErrorResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	// Verify fields
	if unmarshaled.Code != errorResp.Code {
		t.Errorf("Code = %v, want %v", unmarshaled.Code, errorResp.Code)
	}
	if unmarshaled.Message != errorResp.Message {
		t.Errorf("Message = %v, want %v", unmarshaled.Message, errorResp.Message)
	}
	if unmarshaled.Details != errorResp.Details {
		t.Errorf("Details = %v, want %v", unmarshaled.Details, errorResp.Details)
	}
	if unmarshaled.RequestID != errorResp.RequestID {
		t.Errorf("RequestID = %v, want %v", unmarshaled.RequestID, errorResp.RequestID)
	}
}

func TestNewAIRequest(t *testing.T) {
	data := map[string]interface{}{"query": "SELECT * FROM users"}
	request := NewAIRequest("convert_to_sql", data)

	if request.Action != "convert_to_sql" {
		t.Errorf("Action = %v, want %v", request.Action, "convert_to_sql")
	}
	if request.Data == nil {
		t.Error("Data should not be nil")
	}
	if request.RequestID == "" {
		t.Error("RequestID should not be empty")
	}
}

func TestNewAIResponse(t *testing.T) {
	data := map[string]interface{}{"sql": "SELECT * FROM users WHERE id = 1"}
	response := NewAIResponse(true, data, "Success", "test-123")

	if response.Success != true {
		t.Errorf("Success = %v, want %v", response.Success, true)
	}
	if response.Message != "Success" {
		t.Errorf("Message = %v, want %v", response.Message, "Success")
	}
	if response.RequestID != "test-123" {
		t.Errorf("RequestID = %v, want %v", response.RequestID, "test-123")
	}
	if response.Data == nil {
		t.Error("Data should not be nil")
	}
}
