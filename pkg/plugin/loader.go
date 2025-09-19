/*
Copyright 2025 API Testing Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugin

import (
	"context"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoaderMethods implements all the standard testing.Loader interface methods
// These methods provide basic no-op implementations since this is primarily an AI plugin

// ListTestSuite lists all available test suites
func (s *AIPluginService) ListTestSuite(ctx context.Context, req *server.Empty) (*remote.TestSuites, error) {
	// AI plugin doesn't manage test suites directly - return empty list
	return &remote.TestSuites{
		Data: []*remote.TestSuite{},
	}, nil
}

// CreateTestSuite creates a new test suite
func (s *AIPluginService) CreateTestSuite(ctx context.Context, req *remote.TestSuite) (*server.Empty, error) {
	// AI plugin doesn't support creating test suites
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test suite creation")
}

// GetTestSuite retrieves a specific test suite
func (s *AIPluginService) GetTestSuite(ctx context.Context, req *remote.TestSuite) (*remote.TestSuite, error) {
	// AI plugin doesn't manage test suites
	return nil, status.Errorf(codes.NotFound, "test suite not found: %s", req.Name)
}

// UpdateTestSuite updates an existing test suite
func (s *AIPluginService) UpdateTestSuite(ctx context.Context, req *remote.TestSuite) (*remote.TestSuite, error) {
	// AI plugin doesn't support updating test suites
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test suite updates")
}

// DeleteTestSuite deletes a test suite
func (s *AIPluginService) DeleteTestSuite(ctx context.Context, req *remote.TestSuite) (*server.Empty, error) {
	// AI plugin doesn't support deleting test suites
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test suite deletion")
}

// RenameTestSuite renames a test suite
func (s *AIPluginService) RenameTestSuite(ctx context.Context, req *server.TestSuiteDuplicate) (*server.HelloReply, error) {
	// AI plugin doesn't support renaming test suites
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test suite renaming")
}

// ListTestCases lists all test cases in a test suite
func (s *AIPluginService) ListTestCases(ctx context.Context, req *remote.TestSuite) (*server.TestCases, error) {
	// AI plugin doesn't manage test cases directly - return empty list
	return &server.TestCases{
		Data: []*server.TestCase{},
	}, nil
}

// CreateTestCase creates a new test case
func (s *AIPluginService) CreateTestCase(ctx context.Context, req *server.TestCase) (*server.Empty, error) {
	// AI plugin doesn't support creating test cases
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test case creation")
}

// GetTestCase retrieves a specific test case
func (s *AIPluginService) GetTestCase(ctx context.Context, req *server.TestCase) (*server.TestCase, error) {
	// AI plugin doesn't manage test cases
	return nil, status.Errorf(codes.NotFound, "test case not found: %s", req.Name)
}

// UpdateTestCase updates an existing test case
func (s *AIPluginService) UpdateTestCase(ctx context.Context, req *server.TestCase) (*server.TestCase, error) {
	// AI plugin doesn't support updating test cases
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test case updates")
}

// DeleteTestCase deletes a test case
func (s *AIPluginService) DeleteTestCase(ctx context.Context, req *server.TestCase) (*server.Empty, error) {
	// AI plugin doesn't support deleting test cases
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test case deletion")
}

// RenameTestCase renames a test case
func (s *AIPluginService) RenameTestCase(ctx context.Context, req *server.TestCaseDuplicate) (*server.HelloReply, error) {
	// AI plugin doesn't support renaming test cases
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test case renaming")
}

// ListHistoryTestSuite lists history of test suite executions
func (s *AIPluginService) ListHistoryTestSuite(ctx context.Context, req *server.Empty) (*remote.HistoryTestSuites, error) {
	// AI plugin doesn't manage test history - return empty list
	return &remote.HistoryTestSuites{
		Data: []*remote.HistoryTestSuite{},
	}, nil
}

// CreateTestCaseHistory creates a test case execution history entry
func (s *AIPluginService) CreateTestCaseHistory(ctx context.Context, req *server.HistoryTestResult) (*server.Empty, error) {
	// AI plugin doesn't support creating test case history
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test case history creation")
}

// GetHistoryTestCaseWithResult retrieves a test case history with results
func (s *AIPluginService) GetHistoryTestCaseWithResult(ctx context.Context, req *server.HistoryTestCase) (*server.HistoryTestResult, error) {
	// AI plugin doesn't manage test history
	return nil, status.Errorf(codes.NotFound, "test case history not found")
}

// GetHistoryTestCase retrieves a test case history entry
func (s *AIPluginService) GetHistoryTestCase(ctx context.Context, req *server.HistoryTestCase) (*server.HistoryTestCase, error) {
	// AI plugin doesn't manage test history
	return nil, status.Errorf(codes.NotFound, "test case history not found")
}

// DeleteHistoryTestCase deletes a test case history entry
func (s *AIPluginService) DeleteHistoryTestCase(ctx context.Context, req *server.HistoryTestCase) (*server.Empty, error) {
	// AI plugin doesn't support deleting test case history
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test case history deletion")
}

// DeleteAllHistoryTestCase deletes all test case history for a test case
func (s *AIPluginService) DeleteAllHistoryTestCase(ctx context.Context, req *server.HistoryTestCase) (*server.Empty, error) {
	// AI plugin doesn't support deleting test case history
	return nil, status.Errorf(codes.Unimplemented, "AI plugin does not support test case history deletion")
}

// GetTestCaseAllHistory retrieves all history for a test case
func (s *AIPluginService) GetTestCaseAllHistory(ctx context.Context, req *server.TestCase) (*server.HistoryTestCases, error) {
	// AI plugin doesn't manage test history - return empty list
	return &server.HistoryTestCases{
		Data: []*server.HistoryTestCase{},
	}, nil
}

// GetVersion returns the plugin version information
func (s *AIPluginService) GetVersion(ctx context.Context, req *server.Empty) (*server.Version, error) {
	return &server.Version{
		Version: "1.0.0",
		Commit:  "unknown",
		Date:    "unknown",
	}, nil
}

// PProf provides profiling data for performance analysis
func (s *AIPluginService) PProf(ctx context.Context, req *server.PProfRequest) (*server.PProfData, error) {
	// Basic profiling support - could be enhanced with actual pprof integration
	return &server.PProfData{
		Data: []byte("AI plugin profiling data not available"),
	}, nil
}