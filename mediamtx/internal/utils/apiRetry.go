package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

func retryApi(httpMethod string, apiUrl string, body io.Reader, header map[string]string, isFile bool, fileSize int64) ([]byte, error) {
	const defaultDelay = 5 * time.Second
	const defaultMaxRetries = 3

	bgctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < defaultMaxRetries; i++ {

		var ctx context.Context
		var cancel context.CancelFunc

		if isFile {
			// timeout for average network speed 0.5MBps
			ctx, cancel = context.WithTimeout(bgctx, time.Duration((fileSize/1024/1024)*2)*time.Second)
		} else {
			ctx, cancel = context.WithTimeout(bgctx, defaultDelay*4)
		}
		defer cancel()

		onSuccess, response, err := apiCall(ctx, apiUrl, httpMethod, body, header)

		if onSuccess {
			return response, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("retry operation cancelled :: %v", ctx.Err())
		default:
		}
		fmt.Printf("API call failed: %v. Retrying in %v", err, defaultDelay)
		time.Sleep(defaultDelay)
	}

	return nil, errors.New("API call failed after retries")
}

func RetryApi(httpMethod string, apiUrl string, body io.Reader, header map[string]string) ([]byte, error) {
	return retryApi(httpMethod, apiUrl, body, header, false, 0)
}
func RetryApiFile(httpMethod string, apiUrl string, body io.Reader, header map[string]string, fileSize int64) ([]byte, error) {
	return retryApi(httpMethod, apiUrl, body, header, true, fileSize)
}

func apiCall(ctx context.Context, apiUrl string, httpMethod string, body io.Reader, header map[string]string) (bool, []byte, error) {

	fmt.Println("\n API URL :: ", apiUrl)
	fmt.Println("HTTP Method :: ", httpMethod)
	fmt.Println("Body :: ", body)

	req, _ := http.NewRequestWithContext(ctx, httpMethod, apiUrl, body)

	for k := range header {
		req.Header.Set(k, header[k])
	}

	fmt.Println("Headers :: ", req.Header)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return false, nil, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return false, nil, err
	}
	fmt.Printf("Response :: %s\n", resBody)

	return true, resBody, nil
}
