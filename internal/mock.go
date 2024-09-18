package internal

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/joakim-ribier/go-utils/pkg/genericsutil"
	"github.com/joakim-ribier/go-utils/pkg/iosutil"
	"github.com/joakim-ribier/go-utils/pkg/jsonsutil"
	"github.com/joakim-ribier/go-utils/pkg/logsutil"
	"github.com/joakim-ribier/go-utils/pkg/slicesutil"
	"github.com/joakim-ribier/go-utils/pkg/stringsutil"
	"github.com/joakim-ribier/mockapic/pkg"
)

type MockedRequest struct {
	UUID      string
	CreatedAt string
	// payload from request
	Status      int
	ContentType string
	Charset     string
	Headers     map[string]string
	Body        []byte
}

// Equals returns true if the two requests are equal
func (m MockedRequest) Equals(arg MockedRequest) bool {
	return m.Status == arg.Status &&
		m.ContentType == arg.ContentType &&
		m.Charset == arg.Charset &&
		bytes.Equal(m.Body, arg.Body) &&
		reflect.DeepEqual(m.Headers, arg.Headers)
}

type MockedRequestLight struct {
	UUID        string
	CreatedAt   string
	Status      int
	ContentType string
}

type Mocker interface {
	Get(mockId string) (*MockedRequest, error)
	List() ([]MockedRequestLight, error)
	New(params map[string][]string, body []byte) (*string, error)
	Clean(maxLimit int) (int, error)
}

type Mock struct {
	workingDirectory string
	logger           logsutil.Logger
}

func NewMock(workingDirectory string, logger logsutil.Logger) Mock {
	return Mock{
		workingDirectory: workingDirectory,
		logger:           logger.Namespace("mock")}
}

// Get finds the mocked request {mockId} on the storage
func (m Mock) Get(mockId string) (*MockedRequest, error) {
	return get[MockedRequest](m.workingDirectory, mockId, m.logger)
}

func get[T any](workingDirectory, mockId string, logger logsutil.Logger) (*T, error) {
	bytes, err := iosutil.Load(workingDirectory + "/" + mockId + ".json")
	if err != nil {
		logger.Error(err, "error to load data", "mockId", mockId, "workingDirectory", workingDirectory)
		return nil, err
	}

	mock, err := jsonsutil.Unmarshal[T](bytes)
	if err != nil {
		logger.Error(err, "error to unmarshal data", "mockId", mockId, "workingDirectory", workingDirectory, "data", bytes)
		return nil, err
	}
	return &mock, nil
}

// List gets all mocked request on the storage
func (m Mock) List() ([]MockedRequestLight, error) {
	entries, err := os.ReadDir(m.workingDirectory + "/")
	if err != nil {
		m.logger.Error(err, "error to read directory", "workingDirectory", m.workingDirectory)
		return nil, err
	}

	values := slicesutil.SortT[MockedRequestLight, string](
		slicesutil.TransformT[fs.DirEntry, MockedRequestLight](entries, func(e fs.DirEntry) (*MockedRequestLight, error) {
			var mockId string = ""
			if len(e.Name()) > 5 {
				mockId = e.Name()[:len(e.Name())-5]
			}
			return get[MockedRequestLight](m.workingDirectory, mockId, m.logger)
		}), func(mrl1, mrl2 MockedRequestLight) (string, string) { return mrl2.CreatedAt, mrl1.CreatedAt })

	return genericsutil.OrElse(
		values, func() bool { return len(values) > 0 }, []MockedRequestLight{}), nil
}

// New creates a new mocked request and returns the new UUID
func (m Mock) New(reqParams map[string][]string, reqBody []byte) (*string, error) {
	mock := &MockedRequest{
		UUID:      uuid.NewString(),
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		Body:      reqBody,
		Headers:   map[string]string{},
	}

	getReqParam := func(values []string) string {
		if len(values) == 0 {
			return ""
		}
		return values[0]
	}

	for name, values := range reqParams {
		switch name {
		case "contentType":
			mock.ContentType = getReqParam(values)
		case "charset":
			mock.Charset = getReqParam(values)
		case "status":
			mock.Status = stringsutil.Int(getReqParam(values), -1)
		default:
			if len(values) > 0 {
				mock.Headers[name] = values[0]
			}
		}
	}

	if _, is := pkg.HTTP_CODES[mock.Status]; !is {
		return nil, fmt.Errorf("status {%d} does not exist", mock.Status)
	}

	if !slicesutil.Exist(pkg.CONTENT_TYPES, mock.ContentType) {
		return nil, fmt.Errorf("content type {%s} does not exist", mock.ContentType)
	}

	if !slicesutil.Exist(pkg.CHARSET, mock.Charset) {
		return nil, fmt.Errorf("charset {%s} does not exist", mock.Charset)
	}

	reqBody, err := jsonsutil.Marshal(mock)
	if err != nil {
		m.logger.Error(err, "error to nmarshal data", "mock", mock)
		return nil, err
	}

	err = iosutil.Write(reqBody, m.workingDirectory+"/"+mock.UUID+".json")
	if err != nil {
		m.logger.Error(err, "error to write data", "mock", mock, "workingDirectory", m.workingDirectory)
		return nil, err
	}

	return &mock.UUID, nil
}

// Clean removes the x (nb mocked request - max limit) last requests
func (m Mock) Clean(maxLimit int) (int, error) {
	nb := 0
	if maxLimit < 1 {
		return nb, nil
	}
	mockedRequests, err := m.List()
	if err != nil {
		m.logger.Error(err, "error to list requests", "workingDirectory", m.workingDirectory)
		return nb, err
	}
	nbToDelete := len(mockedRequests) - maxLimit
	if nbToDelete < 1 {
		return nb, nil
	}
	for _, mockedRequest := range mockedRequests[len(mockedRequests)-nbToDelete:] {
		if err := os.Remove(m.workingDirectory + "/" + mockedRequest.UUID + ".json"); err == nil {
			nb = nb + 1
		}
	}
	return nb, nil
}
