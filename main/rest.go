package main

import (
	"bufio"
	"errors"
	"io"
	"net/http"
)

const server string = "https://jch.irif.fr:8443/"
const peers string = "peers/"

func getRequest(c *http.Client, URL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", server+peers, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.Do(req)
	return res, err
}

func getPeers(c *http.Client) ([]string, error) {
	res, err := getRequest(c, server+peers)
	if err != nil {
		return nil, err
	}

	arr_peers := make([]string, 1)

	if res.StatusCode == 200 {
		scanner := bufio.NewScanner(res.Body)
		for scanner.Scan() {
			content := scanner.Text()
			if content != "" {
				arr_peers = append(arr_peers, content)
			}
		}

		return arr_peers, nil
	}
	return nil, errors.New("invalid status code")
}

func getAddresses(c *http.Client, peer string) ([]string, error) {
	res, err := getRequest(c, server+peers+peer+"/addresses")
	if err != nil {
		return nil, err
	}

	arr_addr := make([]string, 1)

	if res.StatusCode == 200 {
		scanner := bufio.NewScanner(res.Body)
		for scanner.Scan() {
			content := scanner.Text()
			if content != "" {
				arr_addr = append(arr_addr, content)
			}
		}

		return arr_addr, nil
	}

	if res.StatusCode == 404 {
		return nil, errors.New("unknown peer")
	}

	return nil, errors.New("invalid status code")
}

func getKey(c *http.Client, peer string) ([]byte, error) {
	res, err := getRequest(c, server+peers+peer+"/key")
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		key, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return key, nil
	}
	if res.StatusCode == 204 {
		return nil, errors.New("no key registered")
	}
	if res.StatusCode == 404 {
		return nil, errors.New("unknown peer")
	}

	return nil, errors.New("invalid status code")
}

func getRoot(c *http.Client, peer string) ([]byte, error) {
	res, err := getRequest(c, server+peers+peer+"/root")
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		key, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return key, nil
	}
	if res.StatusCode == 204 {
		return nil, errors.New("no root registered")
	}
	if res.StatusCode == 404 {
		return nil, errors.New("unknown peer")
	}

	return nil, errors.New("invalid status code")

}
