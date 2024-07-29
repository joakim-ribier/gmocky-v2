package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joakim-ribier/gmocky-v2/internal"
	"github.com/joakim-ribier/gmocky-v2/pkg"
	"github.com/joakim-ribier/go-utils/pkg/httpsutil"
	"github.com/joakim-ribier/go-utils/pkg/jsonsutil"
)

type MockerTest struct {
	mockResponse       *internal.MockedRequest
	mockResponseLights []internal.MockedRequestLight
}

// Get finds the mocked request {{mockId}} on the storage.
func (m *MockerTest) Get(mockId string) (*internal.MockedRequest, error) {
	if m.mockResponse != nil {
		return m.mockResponse, nil
	}
	return nil, errors.New("mockId does not exist")
}

// List gets all mocked request on the storage.
func (m *MockerTest) List() ([]internal.MockedRequestLight, error) {
	if m.mockResponseLights != nil {
		return m.mockResponseLights, nil
	}
	return nil, errors.New("error to list mocked responses")
}

// New adds a new mocked request abd returns the new UUID of the mock.
func (m *MockerTest) New(body []byte) (*string, error) {
	mock, err := jsonsutil.Unmarshal[internal.MockedRequest](body)
	if err != nil {
		return nil, errors.New("error to add new mocked response")
	}
	m.mockResponse = &mock
	var r string = "OK"
	return &r, nil
}

// TestListen calls HTTPServer.Listen(),
// checking for a valid return value.
func TestListen(t *testing.T) {
	var askShutdown = false
	httpServer := NewHTTPServer("4333", &MockerTest{})
	defer httpServer.Server.Shutdown(context.Background())

	go func() {
		err := httpServer.Listen()
		if err != nil && !askShutdown {
			t.Errorf("Error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	req, _ := httpsutil.NewHttpRequest("http://localhost:4333/", "")
	resp, _ := req.Call()
	if resp.StatusCode != 200 {
		t.Fatalf(`result: {%v} but expected {%v}`, resp.StatusCode, 200)
	}

	// testing '404' if bad endpoint is called
	req, _ = httpsutil.NewHttpRequest("http://localhost:4333/", "")
	resp, _ = req.Method("POST").Call()
	if resp.StatusCode != 404 {
		t.Fatalf(`result: {%v} but expected {%v}`, resp.StatusCode, 200)
	}

	// shutdown safely the server with defer
	askShutdown = true
}

// ##
// #### ~/ endpoint
// ##

// TestRootEndpoint calls HTTPServer.home(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestRootEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	NewHTTPServer("{port}", &MockerTest{}).home(w, req)

	_, body := geResultResponse(w, t)
	if !strings.Contains(string(body), internal.LOGO) {
		t.Fatalf(`result: {%v} but expected {%v}`, string(body), internal.LOGO)
	}
}

// ##
// #### ~/static/content-types endpoint
// ##

// TestGetContentTypesEndpoint calls HTTPServer.getContentTypes(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestGetContentTypesEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/content-types", nil)
	w := httptest.NewRecorder()

	NewHTTPServer("{port}", &MockerTest{}).getContentTypes(w, req)

	_, body := geResultResponse(w, t)
	if !strings.Contains(string(body), "application/json") {
		t.Fatalf(`result: {%v} but expected {%v}`, string(body), pkg.CONTENT_TYPES)
	}
}

// ##
// #### ~/static/charsets endpoint
// ##

// TestGetCharsetsEndpoint calls HTTPServer.getCharsets(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestGetCharsetsEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/charsets", nil)
	w := httptest.NewRecorder()

	NewHTTPServer("{port}", &MockerTest{}).getCharsets(w, req)

	_, body := geResultResponse(w, t)
	if !strings.Contains(string(body), "ISO-8859-1") {
		t.Fatalf(`result: {%v} but expected {%v}`, string(body), pkg.CHARSET)
	}
}

// ##
// #### ~/static/status-codes endpoint
// ##

// TestGetStatusCodesEndpoint calls HTTPServer.getStatusCodes(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestGetStatusCodesEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/content-types", nil)
	w := httptest.NewRecorder()

	NewHTTPServer("{port}", &MockerTest{}).getStatusCodes(w, req)

	_, body := geResultResponse(w, t)
	if !strings.Contains(string(body), "Method Not Allowed") {
		t.Fatalf(`result: {%v} but expected {%v}`, string(body), pkg.HTTP_CODES)
	}
}

// ##
// #### ~/v1/{uuid} endpoint
// ##

// TestFindMockResponseEndpointWithInvalidUUID calls HTTPServer.findMock(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestFindMockResponseEndpointWithInvalidUUID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/wrong-uuid", nil)
	w := httptest.NewRecorder()

	NewHTTPServer("{port}", &MockerTest{}).findMock(w, req)

	res, body := geResultResponse(w, t)
	if res.Status != "409 Conflict" || string(body) != `{"message": "invalid UUID length: 10"}` {
		t.Fatalf(`result: {%v} but expected {%v}`, res, "409")
	}
}

// TestFindMockResponseEndpointUUIDNotFound calls HTTPServer.findMock(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestFindMockResponseEndpointUUIDNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/"+uuid.NewString(), nil)
	w := httptest.NewRecorder()

	NewHTTPServer("{port}", &MockerTest{}).findMock(w, req)

	res, _ := geResultResponse(w, t)
	if res.Status != "404 Not Found" {
		t.Fatalf(`result: {%v} but expected {%v}`, res, "404")
	}
}

// TestFindMockResponseEndpoint calls HTTPServer.findMock(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestFindMockResponseEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/"+uuid.NewString(), nil)
	w := httptest.NewRecorder()

	mocker := &MockerTest{
		mockResponse: &internal.MockedRequest{
			Status:      200,
			ContentType: "text/plain",
			Charset:     "UTF-8",
			Body:        "Hello World",
		},
	}
	NewHTTPServer("{port}", mocker).findMock(w, req)

	res, _ := geResultResponse(w, t)
	if res.Status != "200 OK" {
		t.Fatalf(`result: {%v} but expected {%v}`, res, mocker)
	}
}

// ##
// #### ~/v1/list endpoint
// ##

// TestListEndpointWithError calls HTTPServer.list(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestListEndpointWithError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/list", nil)
	w := httptest.NewRecorder()

	NewHTTPServer("{port}", &MockerTest{}).list(w, req)

	res, body := geResultResponse(w, t)
	if res.Status != "409 Conflict" || string(body) != `{"message": "error to list mocked responses"}` {
		t.Fatalf(`result: {%v} but expected {%v}`, res, "409")
	}
}

// TestListEndpoint calls HTTPServer.list(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestListEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/list", nil)
	w := httptest.NewRecorder()

	mocker := &MockerTest{
		mockResponseLights: []internal.MockedRequestLight{
			{
				UUID:        uuid.NewString(),
				Status:      200,
				ContentType: "text/plain",
			}, {
				UUID:   uuid.NewString(),
				Status: 404,
			},
		},
	}
	NewHTTPServer("{port}", mocker).list(w, req)

	res, body := geResultResponse(w, t)
	s, err := jsonsutil.Unmarshal[[]internal.MockedRequestLight](body)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if res.Status != "200 OK" || len(s) != 2 {
		t.Fatalf(`result: {%v} but expected {%v}`, res, mocker)
	}
}

// ##
// #### ~/v1/new endpoint
// ##

// TestAddNewEndpoint calls HTTPServer.addNewMock(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestAddNewEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/new", strings.NewReader(`{
    	"status": 200,
    	"contentType": "text/plain",
    	"charset": "UTF-8",
    	"body": "Hello World"
	}`))
	w := httptest.NewRecorder()

	mocker := &MockerTest{mockResponse: nil}
	NewHTTPServer("{port}", mocker).addNewMock(w, req)

	res, body := geResultResponse(w, t)

	expected := &internal.MockedRequest{
		Status:      200,
		ContentType: "text/plain",
		Charset:     "UTF-8",
		Body:        "Hello World",
	}
	if res.Status != "200 OK" ||
		!strings.Contains(string(body), `{"uuid":`) ||
		!mocker.mockResponse.Equals(*expected) {
		t.Fatalf(`result: {%v} but expected {%v}`, res, expected)
	}
}

// TestAddNewEndpointWithBadBody calls HTTPServer.addNewMock(http.ResponseWriter, *http.Request),
// checking for a valid return value.
func TestAddNewEndpointWithBadBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/new", strings.NewReader("bad body..."))
	w := httptest.NewRecorder()

	mocker := &MockerTest{mockResponse: nil}
	NewHTTPServer("{port}", mocker).addNewMock(w, req)

	res, body := geResultResponse(w, t)
	t.Log(string(body))

	if res.Status != "409 Conflict" ||
		string(body) != `{"message": "error to add new mocked response"}` {
		t.Fatalf(`result: {%v} but expected {%v}`, res, "409")
	}
}

func geResultResponse(w *httptest.ResponseRecorder, t *testing.T) (http.Response, []byte) {
	res := w.Result()
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	return *res, data
}
